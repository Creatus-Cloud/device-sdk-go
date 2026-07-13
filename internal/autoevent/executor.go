// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2019-2025 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package autoevent

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/OneOfOne/xxhash"
	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
	"github.com/google/uuid"
	"github.com/panjf2000/ants/v2"

	"github.com/edgexfoundry/device-sdk-go/v4/internal/application"
	sdkCommon "github.com/edgexfoundry/device-sdk-go/v4/internal/common"
	"github.com/edgexfoundry/device-sdk-go/v4/internal/container"

	"github.com/spf13/cast"
)

type Executor struct {
	deviceName        string
	sourceName        string
	onChange          bool
	onChangeThreshold float64
	lastReadings      map[string]any
	onChangeReadings  []dtos.BaseReading
	duration          time.Duration
	stop              bool
	mutex             *sync.Mutex
	pool              *ants.Pool
}

// Run triggers this Executor executes the handler for the event source periodically
func (e *Executor) Run(ctx context.Context, wg *sync.WaitGroup, buffer chan bool, dic *di.Container) {
	wg.Add(1)
	defer wg.Done()

	lc := bootstrapContainer.LoggingClientFrom(dic.Get)
	deadline := time.Now().Add(e.duration)

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Until(deadline)):
			if e.stop {
				return
			}
			deadline = deadline.Add(e.duration)
			lc.Debugf("AutoEvent - reading %s", e.sourceName)
			err := e.Execute(ctx, buffer, dic)
			if err != nil {
				lc.Errorf("AutoEvent - %v", err)
			}
		}
	}
}

func (e *Executor) Execute(ctx context.Context, buffer chan bool, dic *di.Container) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	lc := bootstrapContainer.LoggingClientFrom(dic.Get)
	config := container.ConfigurationFrom(dic.Get)
	evt, err := readResource(ctx, e, dic)
	if err != nil {
		return fmt.Errorf("error occurs when reading resource %s: %w", e.sourceName, err)
	}

	if evt == nil {
		lc.Debugf("AutoEvent - no event generated when reading resource %s", e.sourceName)
		return nil
	}
	if e.onChange {
		if e.compareReadings(evt.Readings) {
			lc.Debugf("AutoEvent - source '%s' readings are the same as previous one", e.sourceName)
			return nil
		}
	}

	// After the auto event executes a read command, it will create a goroutine to send out events.
	// When the concurrent auto event amount becomes large, core-data might be hard to handle so many HTTP requests at the same time.
	// The device service will get some network errors like EOF or Connection reset by peer.
	// By adding a buffer here, the user can use the Service.AsyncBufferSize configuration to control the goroutine for sending events.
	if err := e.pool.Submit(func() {
		buffer <- true
		correlationId := uuid.NewString()

		// Protect e.onChangeReadings with mutex to avoid concurrent access
		e.mutex.Lock()
		if e.onChange && config.Device.AutoEvents.SendChangedReadingsOnly && len(e.onChangeReadings) != 0 {
			// Update the auto event to include only the readings that have changed.
			evt.Readings = e.onChangeReadings
		}
		e.mutex.Unlock()

		sdkCommon.SendEvent(evt, correlationId, dic)
		lc.Tracef("AutoEvent - Sent new Event/Reading for '%s' source with Correlation Id '%s'", evt.SourceName, correlationId)
		<-buffer
	}); err != nil {
		return fmt.Errorf("error occurs when send new event/reading for %s source: %w", e.sourceName, err)
	}
	return nil
}

func readResource(ctx context.Context, e *Executor, dic *di.Container) (event *dtos.Event, err errors.EdgeX) {
	vars := make(map[string]string, 2)
	vars[common.Name] = e.deviceName
	vars[common.Command] = e.sourceName

	res, err := application.GetCommand(ctx, e.deviceName, e.sourceName, "", true, dic)
	if err != nil {
		return event, err
	}
	return res, nil
}

func (e *Executor) compareReadings(readings []dtos.BaseReading) bool {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if len(e.lastReadings) != len(readings) {
		e.renewLastReadings(readings)
		return false
	}

	var result = true
	// Reset the onChangeReadings for each auto event
	e.onChangeReadings = nil

	for _, reading := range readings {
		if lastReading, ok := e.lastReadings[reading.ResourceName]; ok {
			switch reading.ValueType {
			case common.ValueTypeBinary:
				checksum := xxhash.Checksum64(reading.BinaryValue)
				if lastReading != checksum {
					e.lastReadings[reading.ResourceName] = checksum
					result = false
					e.onChangeReadings = append(e.onChangeReadings, reading)
				}
			case common.ValueTypeUint8, common.ValueTypeUint16, common.ValueTypeUint32, common.ValueTypeUint64,
				common.ValueTypeInt8, common.ValueTypeInt16, common.ValueTypeInt32, common.ValueTypeInt64,
				common.ValueTypeFloat32, common.ValueTypeFloat64:
				t := cast.ToFloat64(lastReading) - cast.ToFloat64(reading.Value)
				if math.Abs(t) > e.onChangeThreshold {
					e.lastReadings[reading.ResourceName] = reading.Value
					result = false
					e.onChangeReadings = append(e.onChangeReadings, reading)
				}
			default:
				if lastReading != reading.Value {
					e.lastReadings[reading.ResourceName] = reading.Value
					result = false
					e.onChangeReadings = append(e.onChangeReadings, reading)
				}
			}
		} else {
			e.renewLastReadings(readings)
			return false
		}
	}

	return result
}

func (e *Executor) renewLastReadings(readings []dtos.BaseReading) {
	e.lastReadings = make(map[string]interface{}, len(readings))
	for _, r := range readings {
		if r.ValueType == common.ValueTypeBinary {
			e.lastReadings[r.ResourceName] = xxhash.Checksum64(r.BinaryValue)
		} else {
			e.lastReadings[r.ResourceName] = r.Value
		}
	}
}

// Stop marks this Executor stopped
func (e *Executor) Stop() {
	e.stop = true
}

// NewExecutor creates an Executor for an AutoEvent
func NewExecutor(deviceName string, ae models.AutoEvent, pool *ants.Pool) (*Executor, errors.EdgeX) {
	// check Frequency
	duration, err := time.ParseDuration(ae.Interval)
	if err != nil {
		return nil, errors.NewCommonEdgeX(errors.KindServerError, fmt.Sprintf("failed to parse AutoEvent %s duration", ae.SourceName), err)
	}

	return &Executor{
		deviceName:        deviceName,
		sourceName:        ae.SourceName,
		onChange:          ae.OnChange,
		onChangeThreshold: ae.OnChangeThreshold,
		duration:          duration,
		stop:              false,
		mutex:             &sync.Mutex{},
		pool:              pool,
	}, nil
}

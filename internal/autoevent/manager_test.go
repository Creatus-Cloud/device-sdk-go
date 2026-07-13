// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2026 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package autoevent

import (
	"context"
	"sync"
	"testing"

	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
	"github.com/panjf2000/ants/v2"
	"github.com/stretchr/testify/require"

	"github.com/edgexfoundry/device-sdk-go/v4/pkg/interfaces"
)

type recordingAutoEventScheduler struct {
	started []interfaces.AutoEventContext
	stopped []string
}

func (s *recordingAutoEventScheduler) StartAutoEvents(_ context.Context, _ *sync.WaitGroup, autoEvents []interfaces.AutoEventContext) {
	s.started = append(s.started, autoEvents...)
}

func (s *recordingAutoEventScheduler) StopForDevice(deviceName string) {
	s.stopped = append(s.stopped, deviceName)
}

func TestEventContextsBuildsExecutableContexts(t *testing.T) {
	pool, err := ants.NewPool(1)
	require.NoError(t, err)
	defer pool.Release()

	m := &manager{
		dic:             mockAutoEventDIC(),
		pool:            pool,
		autoeventBuffer: make(chan bool, 1),
	}
	device := models.Device{
		Name:        "device",
		ProfileName: "profile",
		AutoEvents: []models.AutoEvent{
			{SourceName: "Temperature", Interval: "1s"},
		},
	}

	contexts := m.eventContexts(device)

	require.Len(t, contexts, 1)
	require.Equal(t, device.Name, contexts[0].Device.Name)
	require.Equal(t, "Temperature", contexts[0].AutoEvent.SourceName)
	require.NotNil(t, contexts[0].Execute)
}

func TestStopForDeviceDelegatesToCustomScheduler(t *testing.T) {
	scheduler := &recordingAutoEventScheduler{}
	m := &manager{scheduler: scheduler}

	m.StopForDevice("device")

	require.Equal(t, []string{"device"}, scheduler.stopped)
}

func mockAutoEventDIC() *di.Container {
	return di.NewContainer(di.ServiceConstructorMap{
		bootstrapContainer.LoggingClientInterfaceName: func(get di.Get) interface{} {
			return logger.NewMockClient()
		},
	})
}

//
// Copyright (C) 2021 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package interfaces

import (
	"context"
	"sync"

	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
)

type AutoEventManager interface {
	// StartAutoEvents starts all the AutoEvents of the device service
	StartAutoEvents()
	// RestartForDevice restarts all the AutoEvents of the specific device
	RestartForDevice(name string)
	// StopForDevice stops all the AutoEvents of the specific device
	StopForDevice(name string)
}

// AutoEventContext contains the data and execution callback for an AutoEvent.
type AutoEventContext struct {
	Device    models.Device
	AutoEvent models.AutoEvent
	Execute   func(ctx context.Context) error
}

// AutoEventScheduler allows device services to override the default AutoEvent execution scheduling.
type AutoEventScheduler interface {
	// StartAutoEvents starts scheduling the provided AutoEvents.
	StartAutoEvents(ctx context.Context, wg *sync.WaitGroup, autoEvents []AutoEventContext)
	// StopForDevice stops all scheduled AutoEvents for the specified device.
	StopForDevice(deviceName string)
}

// AutoEventSchedulerSetter can be implemented by DeviceServiceSDK instances that support custom AutoEvent schedulers.
type AutoEventSchedulerSetter interface {
	SetAutoEventScheduler(scheduler AutoEventScheduler)
}

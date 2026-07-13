// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2019-2024 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package autoevent

import (
	"context"
	"sync"

	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/startup"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
	"github.com/panjf2000/ants/v2"

	"github.com/edgexfoundry/device-sdk-go/v4/internal/cache"
	"github.com/edgexfoundry/device-sdk-go/v4/internal/container"
	"github.com/edgexfoundry/device-sdk-go/v4/pkg/interfaces"
)

type manager struct {
	executorMap     map[string][]*Executor
	scheduler       interfaces.AutoEventScheduler
	ctx             context.Context
	wg              *sync.WaitGroup
	mutex           sync.Mutex
	autoeventBuffer chan bool
	dic             *di.Container
	pool            *ants.Pool
}

type Bootstrap struct {
	pool *ants.Pool
}

func NewBootstrap(p *ants.Pool) *Bootstrap {
	return &Bootstrap{
		pool: p,
	}
}

func (b *Bootstrap) BootstrapHandler(
	ctx context.Context,
	wg *sync.WaitGroup,
	_ startup.Timer,
	dic *di.Container) bool {
	config := container.ConfigurationFrom(dic.Get)
	m := &manager{
		ctx:             ctx,
		wg:              wg,
		executorMap:     make(map[string][]*Executor),
		dic:             dic,
		autoeventBuffer: make(chan bool, config.Device.AsyncBufferSize),
		pool:            b.pool,
	}

	dic.Update(di.ServiceConstructorMap{
		container.AutoEventManagerName: func(get di.Get) interface{} {
			return m
		},
	})

	return true
}

func (m *manager) StartAutoEvents() {
	m.mutex.Lock()
	scheduler := m.scheduler
	m.mutex.Unlock()

	if scheduler != nil {
		autoEvents := make([]interfaces.AutoEventContext, 0)
		for _, d := range cache.Devices().All() {
			if len(d.ProfileName) == 0 || d.AdminState == models.Locked {
				// don't run the auto event if the device doesn't define the profile, or it is locked
				continue
			}
			autoEvents = append(autoEvents, m.eventContexts(d)...)
		}
		scheduler.StartAutoEvents(m.ctx, m.wg, autoEvents)
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, d := range cache.Devices().All() {
		if len(d.ProfileName) == 0 || d.AdminState == models.Locked {
			// don't run the auto event if the device doesn't define the profile, or it is locked
			continue
		}
		if _, ok := m.executorMap[d.Name]; !ok {
			executors := m.triggerExecutors(d.Name, d.AutoEvents, m.dic)
			m.executorMap[d.Name] = executors
		}
	}
}

func (m *manager) triggerExecutors(deviceName string, autoEvents []models.AutoEvent, dic *di.Container) []*Executor {
	var executors []*Executor
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)

	for _, autoEvent := range autoEvents {
		executor, err := NewExecutor(deviceName, autoEvent, m.pool)
		if err != nil {
			lc.Errorf("failed to create executor of AutoEvent %s for Device %s: %v", autoEvent.SourceName, deviceName, err)
			// skip this AutoEvent if it causes error during creation
			continue
		}
		executors = append(executors, executor)
		go executor.Run(m.ctx, m.wg, m.autoeventBuffer, dic)
	}
	return executors
}

func (m *manager) eventContexts(device models.Device) []interfaces.AutoEventContext {
	lc := bootstrapContainer.LoggingClientFrom(m.dic.Get)
	autoEvents := make([]interfaces.AutoEventContext, 0, len(device.AutoEvents))
	for _, autoEvent := range device.AutoEvents {
		executor, err := NewExecutor(device.Name, autoEvent, m.pool)
		if err != nil {
			lc.Errorf("failed to create executor of AutoEvent %s for Device %s: %v", autoEvent.SourceName, device.Name, err)
			// skip this AutoEvent if it causes error during creation
			continue
		}
		autoEvents = append(autoEvents, interfaces.AutoEventContext{
			Device:    device,
			AutoEvent: autoEvent,
			Execute: func(ctx context.Context) error {
				return executor.Execute(ctx, m.autoeventBuffer, m.dic)
			},
		})
	}
	return autoEvents
}

func (m *manager) RestartForDevice(deviceName string) {
	lc := bootstrapContainer.LoggingClientFrom(m.dic.Get)

	m.StopForDevice(deviceName)
	d, ok := cache.Devices().ForName(deviceName)
	if !ok {
		lc.Errorf("failed to find device %s in cache to start AutoEvent", deviceName)
	}

	if len(d.ProfileName) == 0 || d.AdminState == models.Locked {
		// don't run the auto event if the device doesn't define the profile, or it is locked
		return
	}

	m.mutex.Lock()
	scheduler := m.scheduler
	if scheduler != nil {
		m.mutex.Unlock()
		scheduler.StartAutoEvents(m.ctx, m.wg, m.eventContexts(d))
		return
	}
	executors := m.triggerExecutors(deviceName, d.AutoEvents, m.dic)
	m.executorMap[deviceName] = executors
	m.mutex.Unlock()
}

func (m *manager) StopForDevice(deviceName string) {
	m.mutex.Lock()
	scheduler := m.scheduler
	if scheduler != nil {
		m.mutex.Unlock()
		scheduler.StopForDevice(deviceName)
		return
	}
	defer m.mutex.Unlock()

	executors, ok := m.executorMap[deviceName]
	if ok {
		for _, executor := range executors {
			executor.Stop()
		}
		delete(m.executorMap, deviceName)
	}
}

func (m *manager) SetAutoEventScheduler(scheduler interfaces.AutoEventScheduler) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.scheduler = scheduler
}

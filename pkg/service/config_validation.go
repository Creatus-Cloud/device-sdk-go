package service

import (
	"fmt"
	"net/http"

	"github.com/edgexfoundry/device-sdk-go/v4/internal/config"
	"github.com/edgexfoundry/device-sdk-go/v4/pkg/configvalidation"
	"github.com/edgexfoundry/device-sdk-go/v4/pkg/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos"
	"github.com/labstack/echo/v4"
)

func (s *deviceService) initializeConfigValidation() error {
	s.configValidation = configvalidation.New(s.serviceKey)
	if err := s.configValidation.RegisterStandard(func() any { return &config.ConfigurationStruct{} }, func(raw any) error {
		c := raw.(*config.ConfigurationStruct)
		if err := configvalidation.ValidateService(c.Service, c.Writable.LogLevel); err != nil {
			return err
		}
		if err := configvalidation.ValidateTelemetry(c.Writable.Telemetry); err != nil {
			return err
		}
		// Autoevents also allocate this buffer when asynchronous readings are disabled.
		if c.Device.AsyncBufferSize < 0 {
			return fmt.Errorf("Device.AsyncBufferSize must not be negative")
		}
		if c.MaxEventSize < 0 {
			return fmt.Errorf("MaxEventSize must not be negative")
		}
		return nil
	}); err != nil {
		return err
	}
	s.configValidation.SetDeviceValidator(func(devices []dtos.Device) error {
		for _, device := range devices {
			if err := s.driver.ValidateDevice(dtos.ToDeviceModel(device)); err != nil {
				return fmt.Errorf("device %s: %w", device.Name, err)
			}
		}
		return nil
	})
	return s.AddCustomRoute(configvalidation.Route, interfaces.Authenticated, echo.WrapHandler(s.configValidation), http.MethodGet, http.MethodPost)
}

// RegisterConfigValidator adds an isolated configuration factory and validator.
// It is an optional extension and does not change DeviceServiceSDK.
func (s *deviceService) RegisterConfigValidator(factory func() any, validate func(any) error) error {
	if s.configValidation == nil {
		return fmt.Errorf("configuration validation is not initialized")
	}
	return s.configValidation.Register(factory, validate)
}

var _ interfaces.ConfigValidationRegistrar = (*deviceService)(nil)

// RegisterCustomConfigValidator registers an existing validator for the named
// custom section. The prototype has the same outer shape as LoadCustomConfig.
// Validation receives a fresh configuration and must not change live state.
func (s *deviceService) RegisterCustomConfigValidator(prototype any, sectionName string, validate func(any) error) error {
	if s.configValidation == nil {
		return fmt.Errorf("configuration validation is not initialized")
	}
	return s.configValidation.RegisterCustom(prototype, sectionName, validate)
}

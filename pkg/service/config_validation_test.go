package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	restController "github.com/edgexfoundry/device-sdk-go/v4/internal/controller/http"
	"github.com/edgexfoundry/device-sdk-go/v4/pkg/configvalidation"
	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func newPreflightTestService(t *testing.T) *deviceService {
	t.Helper()
	t.Setenv("EDGEX_DISABLE_JWT_VALIDATION", "true")
	dic := di.NewContainer(di.ServiceConstructorMap{
		bootstrapContainer.LoggingClientInterfaceName: func(di.Get) interface{} { return logger.NewMockClient() },
	})
	sdk := deviceService{
		serviceKey: "test-service",
		dic:        dic,
		controller: restController.NewRestController(echo.New(), dic, "test-service"),
	}
	require.NoError(t, sdk.initializeConfigValidation())
	return &sdk
}

func TestConfigPreflightTelemetryInterval(t *testing.T) {
	sdk := newPreflightTestService(t)
	for _, test := range []struct {
		name     string
		interval string
		valid    bool
	}{
		{"disabled by empty interval", "", true},
		{"disabled by zero", "0", true},
		{"disabled by zero duration", "0s", true},
		{"positive", "30s", true},
		{"malformed", "not-a-duration", false},
		{"negative", "-1s", false},
		{"overflow", "999999999999999999999h", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := sdk.configValidation.Validate(configvalidation.Request{
				ServiceID: "test-service",
				Configuration: map[string]any{
					"Service/Host":                "localhost",
					"Service/Port":                "59701",
					"Service/RequestTimeout":      "5s",
					"Writable/LogLevel":           "INFO",
					"Writable/Telemetry/Interval": test.interval,
				},
			})
			require.Equal(t, test.valid, result.Valid, "%+v", result.Errors)
			if !test.valid {
				require.Contains(t, fmt.Sprint(result.Errors), "Writable.Telemetry.Interval")
			}
		})
	}
}

func TestConfigPreflightAsyncBufferSize(t *testing.T) {
	sdk := newPreflightTestService(t)
	for _, test := range []struct {
		name, enabled, size string
		valid               bool
	}{
		{"buffered", "true", "16", true},
		{"unbuffered", "true", "0", true},
		{"negative", "true", "-1", false},
		{"disabled buffered", "false", "16", true},
		{"disabled unbuffered", "false", "0", true},
		{"disabled negative", "false", "-1", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := sdk.configValidation.Validate(configvalidation.Request{
				ServiceID: "test-service",
				Configuration: map[string]any{
					"Service/Host": "localhost", "Service/Port": "59701",
					"Service/RequestTimeout": "5s", "Writable/LogLevel": "INFO",
					"Device/EnableAsyncReadings": test.enabled,
					"Device/AsyncBufferSize":     test.size,
				},
			})
			require.Equal(t, test.valid, result.Valid, "%+v", result.Errors)
			if !test.valid {
				require.Contains(t, fmt.Sprint(result.Errors), "Device.AsyncBufferSize")
			}
		})
	}
}

func TestInitializedCapabilitiesShareOnlyStandardConfiguration(t *testing.T) {
	sdk := newPreflightTestService(t)
	require.NoError(t, sdk.RegisterCustomConfigValidator(&struct{ AppCustom struct{ Limit int } }{}, "AppCustom", func(any) error { return nil }))
	response := httptest.NewRecorder()
	sdk.configValidation.ServeHTTP(response, httptest.NewRequest(http.MethodGet, configvalidation.Route, nil))
	var capabilities configvalidation.Capabilities
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &capabilities))
	require.Contains(t, capabilities.Sections, "AppCustom")
	require.NotContains(t, capabilities.SharedSections, "AppCustom")
	for _, section := range []string{"Service", "Writable", "Device"} {
		require.Contains(t, capabilities.SharedSections, section)
	}
}

package service

import (
	"fmt"
	"testing"

	restController "github.com/edgexfoundry/device-sdk-go/v4/internal/controller/http"
	"github.com/edgexfoundry/device-sdk-go/v4/pkg/configvalidation"
	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestConfigPreflightTelemetryInterval(t *testing.T) {
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
				Configuration: map[string]string{
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

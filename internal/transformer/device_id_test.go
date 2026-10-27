// SPDX-License-Identifier: Apache-2.0

package transformer

import (
	"encoding/json"
	"testing"

	"github.com/edgexfoundry/device-sdk-go/v4/internal/cache"
	sdkModels "github.com/edgexfoundry/device-sdk-go/v4/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos"
	"github.com/fxamacker/cbor/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandValuesToEventDTO_DeviceID(t *testing.T) {
	const origin int64 = 1793098800123456789
	dic := NewMockDIC()
	require.NoError(t, cache.InitCache(TestDeviceService, TestDeviceService, dic))
	profile, exists := cache.Profiles().ForName(TestProfile)
	require.True(t, exists)
	resource := profile.DeviceResources[0]
	profile.DeviceResources = nil
	profile.DeviceCommands = nil

	cases := []struct {
		name      string
		valueType string
		value     any
	}{
		{"string", common.ValueTypeString, "on"},
		{"integer", common.ValueTypeInt64, int64(42)},
		{"float", common.ValueTypeFloat64, float64(12.5)},
		{"boolean", common.ValueTypeBool, true},
		{"array", common.ValueTypeStringArray, []string{"one", "two"}},
		{"binary", common.ValueTypeBinary, []byte("01234567890123456789")},
		{"object", common.ValueTypeObject, map[string]any{"state": "on"}},
		{"object_array", common.ValueTypeObjectArray, []map[string]any{{"state": "off"}}},
		{"null", common.ValueTypeString, nil},
	}
	var values []*sdkModels.CommandValue
	for _, tc := range cases {
		dr := resource
		dr.Name = tc.name
		dr.Properties.ValueType = tc.valueType
		dr.Properties.MediaType = "application/octet-stream"
		dr.Tags = map[string]any{"DEVICE_ID": "configured-id", "label": tc.name}
		profile.DeviceResources = append(profile.DeviceResources, dr)
		cv, err := sdkModels.NewCommandValueWithOrigin(tc.name, tc.valueType, tc.value, origin)
		require.NoError(t, err)
		cv.Tags["source"] = "driver"
		values = append(values, cv)
	}
	require.NoError(t, cache.Profiles().Update(profile))
	event, err := CommandValuesToEventDTO(values, TestDevice, "all-readings", false, dic)
	require.NoError(t, err)
	require.NotNil(t, event)
	require.Len(t, event.Readings, len(cases))
	assert.Equal(t, "driver", event.Tags["source"])
	for i, reading := range event.Readings {
		assert.Equal(t, TestDeviceID, reading.DeviceID)
		assert.Equal(t, "configured-id", reading.Tags["DEVICE_ID"], "profile tags must remain independent")
		assert.Empty(t, reading.Extensions)
		assert.Equal(t, cases[i].name, reading.Tags["label"])
		assert.Equal(t, cases[i].valueType, reading.ValueType)
		assert.Equal(t, TestDevice, reading.DeviceName)
		assert.Equal(t, origin, reading.Origin)
		assert.NotEmpty(t, reading.Id)
		assert.NotEqual(t, TestDeviceID, reading.Id, "reading identity must remain distinct")
	}
	// Events travel as either JSON or CBOR; the ID must survive both encodings.
	for _, encoding := range []struct {
		name      string
		marshal   func(any) ([]byte, error)
		unmarshal func([]byte, any) error
	}{
		{"json", json.Marshal, json.Unmarshal},
		{"cbor", cbor.Marshal, cbor.Unmarshal},
	} {
		t.Run(encoding.name, func(t *testing.T) {
			payload, err := encoding.marshal(event)
			require.NoError(t, err)
			var wire struct {
				Readings []struct {
					DeviceID string `json:"deviceID" cbor:"deviceID"`
					Origin   int64  `json:"origin" cbor:"origin"`
				} `json:"readings" cbor:"readings"`
			}
			require.NoError(t, encoding.unmarshal(payload, &wire))
			require.Len(t, wire.Readings, len(cases))
			for _, reading := range wire.Readings {
				assert.Equal(t, TestDeviceID, reading.DeviceID, "deviceID must be a top-level reading field")
				assert.Equal(t, origin, reading.Origin, "serialization must preserve nanosecond precision")
			}
			var decoded dtos.Event
			require.NoError(t, encoding.unmarshal(payload, &decoded))
			require.Len(t, decoded.Readings, len(cases))
			for i, reading := range decoded.Readings {
				assert.Equal(t, TestDeviceID, reading.DeviceID)
				assert.NotContains(t, reading.Extensions, "deviceID")
				assert.Equal(t, cases[i].name, reading.Tags["label"])
				assert.Equal(t, event.Readings[i].Id, reading.Id)
				assert.Equal(t, event.Readings[i].Origin, reading.Origin)
			}
		})
	}
	// Building readings must not overwrite tags in the shared profile cache.
	cached, exists := cache.Profiles().ForName(TestProfile)
	require.True(t, exists)
	for _, dr := range cached.DeviceResources {
		assert.Equal(t, "configured-id", dr.Tags["DEVICE_ID"])
	}
}

func TestCommandValuesToEventDTO_DeviceIDSurvivesRename(t *testing.T) {
	dic := NewMockDIC()
	require.NoError(t, cache.InitCache(TestDeviceService, TestDeviceService, dic))
	cv, err := sdkModels.NewCommandValue(TestDeviceResource, common.ValueTypeString, TestValue)
	require.NoError(t, err)
	before, err := CommandValuesToEventDTO([]*sdkModels.CommandValue{cv}, TestDevice, TestDeviceResource, false, dic)
	require.NoError(t, err)
	device, exists := cache.Devices().ForName(TestDevice)
	require.True(t, exists)
	require.NoError(t, cache.Devices().RemoveByName(TestDevice))
	device.Name = "renamed-device"
	require.NoError(t, cache.Devices().Add(device))
	after, err := CommandValuesToEventDTO([]*sdkModels.CommandValue{cv}, device.Name, TestDeviceResource, false, dic)
	require.NoError(t, err)
	// Reusing the old name must not change the identity of earlier readings.
	device.Name, device.Id = TestDevice, "1313aa47-9878-426d-bc1a-d23fb69a7c0d"
	require.NoError(t, cache.Devices().Add(device))
	replacement, err := CommandValuesToEventDTO([]*sdkModels.CommandValue{cv}, TestDevice, TestDeviceResource, false, dic)
	require.NoError(t, err)
	assert.Equal(t, TestDeviceID, before.Readings[0].DeviceID)
	assert.Equal(t, TestDeviceID, after.Readings[0].DeviceID)
	assert.NotContains(t, before.Readings[0].Tags, "DEVICE_ID")
	assert.NotContains(t, after.Readings[0].Tags, "DEVICE_ID")
	assert.Equal(t, "renamed-device", after.Readings[0].DeviceName)
	assert.Equal(t, device.Id, replacement.Readings[0].DeviceID)
}

func TestCommandValuesToEventDTO_MissingDeviceIDDoesNotUseProfileTag(t *testing.T) {
	dic := NewMockDIC()
	require.NoError(t, cache.InitCache(TestDeviceService, TestDeviceService, dic))
	device, exists := cache.Devices().ForName(TestDevice)
	require.True(t, exists)
	device.Id = ""
	require.NoError(t, cache.Devices().Update(device))
	profile, exists := cache.Profiles().ForName(TestProfile)
	require.True(t, exists)
	profile.DeviceResources[0].Tags = map[string]any{"DEVICE_ID": "configured-id", "label": "keep"}
	require.NoError(t, cache.Profiles().Update(profile))
	cv, err := sdkModels.NewCommandValue(TestDeviceResource, common.ValueTypeString, TestValue)
	require.NoError(t, err)
	event, err := CommandValuesToEventDTO([]*sdkModels.CommandValue{cv}, TestDevice, TestDeviceResource, false, dic)
	require.NoError(t, err)
	require.Len(t, event.Readings, 1)
	assert.Empty(t, event.Readings[0].DeviceID)
	assert.Equal(t, "configured-id", event.Readings[0].Tags["DEVICE_ID"])
	assert.Equal(t, "keep", event.Readings[0].Tags["label"])
}

package transformer

import (
	"testing"

	"github.com/edgexfoundry/device-sdk-go/v4/pkg/models"
	"github.com/edgexfoundry/device-sdk-go/v4/pkg/transformhook"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	contractModels "github.com/edgexfoundry/go-mod-core-contracts/v4/models"
)

func TestReadPropertiesHookCanNormalizeLegacyShift(t *testing.T) {
	t.Cleanup(func() { transformhook.SetReadPropertiesHook(nil) })
	transformhook.SetReadPropertiesHook(func(_, _ string, properties contractModels.ResourceProperties) contractModels.ResourceProperties {
		if properties.Shift != nil && *properties.Shift < 0 {
			shift := -*properties.Shift
			properties.Shift = &shift
		}
		return properties
	})

	mask := uint64(0x100)
	legacyShift := int64(-8)
	properties := transformhook.ApplyReadProperties("Evco-EVJ705", "Output1", contractModels.ResourceProperties{
		Mask:  &mask,
		Shift: &legacyShift,
	})
	value, err := models.NewCommandValue("Output1", common.ValueTypeUint16, uint16(0x100))
	if err != nil {
		t.Fatal(err)
	}
	if err := TransformReadResult(value, properties); err != nil {
		t.Fatal(err)
	}
	got, err := value.Uint16Value()
	if err != nil {
		t.Fatal(err)
	}
	if got != 1 {
		t.Fatalf("Output1 = %d, want 1", got)
	}
}

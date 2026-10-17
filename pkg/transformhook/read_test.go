package transformhook

import (
	"testing"

	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
)

func TestApplyReadProperties(t *testing.T) {
	t.Cleanup(func() { SetReadPropertiesHook(nil) })
	shift := int64(-8)
	properties := models.ResourceProperties{Shift: &shift}

	if got := ApplyReadProperties("profile", "resource", properties); got.Shift == nil || *got.Shift != -8 {
		t.Fatalf("default shift = %v, want -8", got.Shift)
	}

	SetReadPropertiesHook(func(_, _ string, properties models.ResourceProperties) models.ResourceProperties {
		normalized := int64(8)
		properties.Shift = &normalized
		return properties
	})
	if got := ApplyReadProperties("profile", "resource", properties); got.Shift == nil || *got.Shift != 8 {
		t.Fatalf("hooked shift = %v, want 8", got.Shift)
	}
}

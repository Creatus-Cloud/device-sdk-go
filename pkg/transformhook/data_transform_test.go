package transformhook

import "testing"

func TestApplyDataTransformOverride(t *testing.T) {
	t.Cleanup(func() { SetDataTransformOverride(false) })

	if ApplyDataTransformOverride(false) {
		t.Fatal("disabled override changed configured false to true")
	}
	if !ApplyDataTransformOverride(true) {
		t.Fatal("disabled override changed configured true to false")
	}

	SetDataTransformOverride(true)
	if !ApplyDataTransformOverride(false) {
		t.Fatal("enabled override did not force configured false to true")
	}
	if !ApplyDataTransformOverride(true) {
		t.Fatal("enabled override changed configured true to false")
	}
}

package transformhook

import "sync/atomic"

var forceDataTransform atomic.Bool

// SetDataTransformOverride forces the SDK's standard read and write
// transformations on when enabled. Disabled preserves the configured
// Device/DataTransform value and therefore the default SDK behavior.
//
// This is intended for services that must remain compatible with profiles
// containing transformation properties even when an older private
// configuration has persisted Device/DataTransform=false.
func SetDataTransformOverride(enabled bool) {
	forceDataTransform.Store(enabled)
}

// ApplyDataTransformOverride returns the effective DataTransform setting.
func ApplyDataTransformOverride(configured bool) bool {
	return configured || forceDataTransform.Load()
}

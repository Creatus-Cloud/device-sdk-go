package transformhook

import (
	"sync"

	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
)

// ReadPropertiesHook may adjust resource properties immediately before the
// SDK applies its standard read transformation.
type ReadPropertiesHook func(profileName, resourceName string, properties models.ResourceProperties) models.ResourceProperties

var (
	readPropertiesMu   sync.RWMutex
	readPropertiesHook ReadPropertiesHook
)

// SetReadPropertiesHook registers the service-specific read-properties hook.
// Passing nil restores the default SDK behavior.
func SetReadPropertiesHook(hook ReadPropertiesHook) {
	readPropertiesMu.Lock()
	readPropertiesHook = hook
	readPropertiesMu.Unlock()
}

// ApplyReadProperties applies the registered hook, if any.
func ApplyReadProperties(profileName, resourceName string, properties models.ResourceProperties) models.ResourceProperties {
	readPropertiesMu.RLock()
	hook := readPropertiesHook
	readPropertiesMu.RUnlock()
	if hook == nil {
		return properties
	}
	return hook(profileName, resourceName, properties)
}

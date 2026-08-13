package inventory

import (
	"fmt"
	"sync"

	utilversion "k8s.io/apimachinery/pkg/util/version"
)

var (
	// registry holds all registered Kubernetes API inventories by version string (e.g., "1.36")
	registry     = make(map[string][]ServedAPIEntry)
	registryLock sync.RWMutex
)

// RegisterKubernetesAPIs registers the API inventory for a specific Kubernetes version.
// This is called by init() functions in generated files.
// The version string should be in the format "1.36".
func RegisterKubernetesAPIs(version string, apis []ServedAPIEntry) {
	registryLock.Lock()
	defer registryLock.Unlock()
	registry[version] = apis
}

// ForKubeVersion returns the Kubernetes API inventory for the given version.
// Returns found=false if the version is not in the registered inventory.
func ForKubeVersion(v *utilversion.Version) ([]ServedAPIEntry, bool) {
	if v.Major() != 1 {
		return nil, false
	}

	key := fmt.Sprintf("%d.%d", v.Major(), v.Minor())

	registryLock.RLock()
	defer registryLock.RUnlock()

	apis, found := registry[key]
	return apis, found
}

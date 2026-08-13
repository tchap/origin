package inventory

import (
	"sync"

	utilversion "k8s.io/apimachinery/pkg/util/version"
)

var (
	// registry holds all registered Kubernetes API inventories by version
	registry = make(map[uint][]ServedAPIEntry)
	mu       sync.RWMutex
)

// RegisterKubernetesAPIs registers the API inventory for a specific Kubernetes minor version.
// This is called by init() functions in generated files.
func RegisterKubernetesAPIs(minor uint, apis []ServedAPIEntry) {
	mu.Lock()
	defer mu.Unlock()
	registry[minor] = apis
}

// ForKubeVersion returns the Kubernetes API inventory for the given version.
// Returns found=false if the version is not in the registered inventory.
func ForKubeVersion(v *utilversion.Version) ([]ServedAPIEntry, bool) {
	if v.Major() != 1 {
		return nil, false
	}

	mu.RLock()
	defer mu.RUnlock()

	apis, found := registry[v.Minor()]
	return apis, found
}

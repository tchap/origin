package inventory

import "github.com/openshift/api/servedapis"

// Re-export types from servedapis for convenience
type ServedAPIEntry = servedapis.ServedAPIEntry
type ClusterProfile = servedapis.ClusterProfile

const (
	ClusterProfileSelfManagedHA = servedapis.ClusterProfileSelfManagedHA
	ClusterProfileHyperShift    = servedapis.ClusterProfileHyperShift
)

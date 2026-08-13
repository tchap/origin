package inventory

// ClusterProfile represents the cluster deployment topology.
type ClusterProfile string

const (
	ClusterProfileSelfManagedHA ClusterProfile = "SelfManagedHA"
	ClusterProfileHypershift    ClusterProfile = "Hypershift"
)

// ServedAPIEntry describes a single API resource that should be served by the cluster.
type ServedAPIEntry struct {
	Group    string
	Version  string
	Resource string
	Kind     string
	Scope    string // "Namespaced" or "Cluster"
	Source   string // informational: "core-kube", "openshift-crd", "openshift-apiserver", etc.
}

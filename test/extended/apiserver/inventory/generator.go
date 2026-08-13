package inventory

import (
	"fmt"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/kubernetes/pkg/controlplane"
)

// nonResourceKinds are Kubernetes types that exist in the client-go scheme but are not
// top-level REST resources: subresource-only types, discovery meta types, and internal
// streaming/status types. This list is small and stable across releases.
var nonResourceKinds = sets.New(
	// subresource-only types (accessible only under a parent resource path)
	"Eviction",
	"NodeProxyOptions",
	"PodAttachOptions",
	"PodExecOptions",
	"PodPortForwardOptions",
	"PodProxyOptions",
	"RangeAllocation",
	"Scale",
	"SerializedReference",
	"ServiceProxyOptions",
	"TokenRequest",
	// discovery meta types (served at /api and /apis, not as regular resources)
	"APIGroup",
	"APIVersions",
	// internal streaming type used for watch responses, not a REST resource
	"WatchEvent",
	// error/status envelope, not a REST resource
	"Status",
	// internal pod phase type, not served
	"PodStatusResult",
)

func shouldSkipKind(kind string) bool {
	if strings.HasSuffix(kind, "List") {
		return true
	}
	if strings.HasSuffix(kind, "Options") {
		return true
	}
	// Note: do NOT filter on "Review" suffix. TokenReview, SubjectAccessReview,
	// SelfSubjectAccessReview etc. are real top-level POST endpoints.
	return nonResourceKinds.Has(kind)
}

// GenerateKubernetesInventory derives the complete set of Kubernetes API resources
// from the vendored scheme and DefaultAPIResourceConfigSource.
func GenerateKubernetesInventory() ([]ServedAPIEntry, error) {
	resourceConfig := controlplane.DefaultAPIResourceConfigSource()
	result := []ServedAPIEntry{}

	for gv, enabled := range resourceConfig.GroupVersionConfigs {
		if !enabled {
			continue
		}
		for kind := range clientgoscheme.Scheme.KnownTypes(gv) {
			if shouldSkipKind(kind) {
				continue
			}
			plural, _ := meta.UnsafeGuessKindToResource(gv.WithKind(kind))

			// Infer scope from the scheme
			scope := inferScope(gv, kind)

			result = append(result, ServedAPIEntry{
				Group:    plural.Group,
				Version:  plural.Version,
				Resource: plural.Resource,
				Kind:     kind,
				Scope:    scope,
				Source:   "core-kube",
			})
		}
	}

	// A small number of required Kubernetes resources live in separate vendored packages
	// (apiextensions-apiserver, kube-aggregator) and are not registered in clientgoscheme.
	for _, entry := range []ServedAPIEntry{
		{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions", Kind: "CustomResourceDefinition", Scope: "Cluster", Source: "core-kube"},
		{Group: "apiregistration.k8s.io", Version: "v1", Resource: "apiservices", Kind: "APIService", Scope: "Cluster", Source: "core-kube"},
	} {
		result = append(result, entry)
	}

	// Sort by (Group, Version, Resource) for stable diffs
	sort.Slice(result, func(i, j int) bool {
		if result[i].Group != result[j].Group {
			return result[i].Group < result[j].Group
		}
		if result[i].Version != result[j].Version {
			return result[i].Version < result[j].Version
		}
		return result[i].Resource < result[j].Resource
	})

	return result, nil
}

// inferScope attempts to determine if a resource is Namespaced or Cluster-scoped.
// This is a best-effort heuristic based on common patterns.
func inferScope(gv schema.GroupVersion, kind string) string {
	// Try to get the object from the scheme to check if it's namespaced
	obj, err := clientgoscheme.Scheme.New(gv.WithKind(kind))
	if err != nil {
		// Fallback to heuristics for common patterns
		return inferScopeFromKind(kind)
	}

	// Check if the object has GetNamespace method (namespaced objects do)
	if ns, ok := obj.(interface{ GetNamespace() string }); ok && ns != nil {
		return "Namespaced"
	}

	return "Cluster"
}

// inferScopeFromKind uses heuristics to guess scope when scheme lookup fails.
func inferScopeFromKind(kind string) string {
	// Cluster-scoped resources (common patterns)
	clusterScoped := sets.New(
		"Node",
		"Namespace",
		"PersistentVolume",
		"ClusterRole",
		"ClusterRoleBinding",
		"StorageClass",
		"VolumeAttachment",
		"CSIDriver",
		"CSINode",
		"CSIStorageCapacity",
		"IngressClass",
		"RuntimeClass",
		"PriorityClass",
		"APIService",
		"CustomResourceDefinition",
		"MutatingWebhookConfiguration",
		"ValidatingWebhookConfiguration",
		"ValidatingAdmissionPolicy",
		"ValidatingAdmissionPolicyBinding",
		"MutatingAdmissionPolicy",
		"MutatingAdmissionPolicyBinding",
		"CertificateSigningRequest",
		"FlowSchema",
		"PriorityLevelConfiguration",
		"IPAddress",
		"ServiceCIDR",
	)

	if clusterScoped.Has(kind) {
		return "Cluster"
	}

	// Default to Namespaced for most resources
	return "Namespaced"
}

// FormatGoCode formats the inventory as Go code for a version-specific generated file.
// The file registers itself via init() so no central switch statement needs updating.
func FormatGoCode(entries []ServedAPIEntry, kubeMinorVersion int) (string, error) {
	var b strings.Builder

	fmt.Fprintf(&b, "// Code generated by test/extended/apiserver/inventory/write-kube-api-inventory. DO NOT EDIT.\n\n")
	fmt.Fprintf(&b, "package inventory\n\n")

	// Generate init function that registers this version (at the top for visibility)
	fmt.Fprintf(&b, "func init() {\n")
	fmt.Fprintf(&b, "\tRegisterKubernetesAPIs(\"1.%d\", kubeAPIs1%d)\n", kubeMinorVersion, kubeMinorVersion)
	fmt.Fprintf(&b, "}\n\n")

	// Generate the variable for this Kubernetes version
	fmt.Fprintf(&b, "var kubeAPIs1%d = []ServedAPIEntry{\n", kubeMinorVersion)
	for _, e := range entries {
		fmt.Fprintf(&b, "\t{Group: %q, Version: %q, Resource: %q, Kind: %q, Scope: %q, Source: %q},\n",
			e.Group, e.Version, e.Resource, e.Kind, e.Scope, e.Source)
	}
	fmt.Fprintf(&b, "}\n")

	return b.String(), nil
}

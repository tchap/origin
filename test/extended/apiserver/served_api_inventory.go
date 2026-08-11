package apiserver

// Proof of concept for the served API inventory e2e test.
// Stubs for the openshift/api servedapis and features packages live in
// served_api_inventory_stubs.go and will be replaced once that package is vendored.

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"

	"github.com/blang/semver/v4"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/discovery"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/kubernetes/pkg/controlplane"
	"k8s.io/kubernetes/test/e2e/framework"

	configv1 "github.com/openshift/api/config/v1"

	exutil "github.com/openshift/origin/test/extended/util"
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

var _ = g.Describe("[sig-api-machinery][Suite:openshift/conformance/parallel] Served API inventory", func() {
	defer g.GinkgoRecover()
	oc := exutil.NewCLIWithoutNamespace("served-api-inventory")

	g.It("should match the expected list", func() {
		ctx := context.Background()

		// 1. Detect cluster state.
		topology, err := exutil.GetControlPlaneTopology(oc)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(topology).NotTo(o.BeNil())

		featureGate, err := oc.AdminConfigClient().ConfigV1().FeatureGates().Get(ctx, "cluster", metav1.GetOptions{})
		o.Expect(err).NotTo(o.HaveOccurred())

		profile := clusterProfileName(*topology)
		featureSet := string(featureGate.Spec.FeatureSet)
		enabledGates := collectEnabledGates(featureGate)

		// Get the Kubernetes version the cluster is actually running.
		// ParseTolerant handles the "v" prefix and any "+openshift" build metadata.
		serverVersion, err := oc.AdminKubeClient().Discovery().ServerVersion()
		o.Expect(err).NotTo(o.HaveOccurred())
		kubeVersion, err := semver.ParseTolerant(serverVersion.GitVersion)
		o.Expect(err).NotTo(o.HaveOccurred())

		framework.Logf("cluster profile=%s featureSet=%q kubeVersion=%s enabledGates=%v",
			profile, featureSet, kubeVersion, enabledGates)

		// 2. Build the OpenShift expected API set (stub; will use vendored servedapis package).
		openshiftEntries := servedAPIsForProfile(profile, featureSet)

		// 3. Build the Kubernetes stable expected API set from the vendored scheme.
		expectedKube, err := kubeResourcesFromScheme()
		o.Expect(err).NotTo(o.HaveOccurred())

		// 4. Apply Kubernetes API overrides enabled by active OpenShift feature gates.
		// The override map is version-aware: a gate may enable different GVRs on different
		// Kubernetes versions (e.g. MutatingAdmissionPolicy at v1alpha1 on kube 1.33,
		// v1beta1 on 1.34+).
		activeOverrides := kubeAPIOverridesByVersion(kubeVersion)
		for _, gate := range enabledGates {
			for _, gvr := range activeOverrides[gate] {
				expectedKube.Insert(gvr)
			}
		}

		// 5. Merge into required and optional sets.
		required := sets.New[schema.GroupVersionResource]()
		optional := sets.New[schema.GroupVersionResource]()

		for _, e := range openshiftEntries {
			gvr := schema.GroupVersionResource{Group: e.Group, Version: e.Version, Resource: e.Resource}
			if e.Source == sourceOptional {
				optional.Insert(gvr)
			} else {
				required.Insert(gvr)
			}
		}
		for gvr := range expectedKube {
			required.Insert(gvr)
		}

		// 6. Query the cluster's actual served APIs via discovery.
		actual, err := discoveredResources(oc)
		o.Expect(err).NotTo(o.HaveOccurred())

		framework.Logf("expected required=%d optional=%d actual=%d", required.Len(), optional.Len(), actual.Len())

		// Log optional API presence for visibility (not a failure either way).
		for _, gvr := range optional.Intersection(actual).UnsortedList() {
			framework.Logf("optional API served: %s", gvrString(gvr))
		}
		for _, gvr := range optional.Difference(actual).UnsortedList() {
			framework.Logf("optional API absent (OK): %s", gvrString(gvr))
		}

		// 7. Bidirectional comparison.
		missing := required.Difference(actual).UnsortedList()
		unexpected := actual.Difference(required.Union(optional)).UnsortedList()

		o.Expect(missing).To(o.BeEmpty(),
			"required APIs not served by the cluster:\n%s", formatGVRList(missing))
		o.Expect(unexpected).To(o.BeEmpty(),
			"APIs served by the cluster but not in the expected list — add them to the servedapis stub or the optional list:\n%s", formatGVRList(unexpected))
	})
})

// clusterProfileName maps the infrastructure topology to the profile name used by the
// servedapis package.
func clusterProfileName(topology configv1.TopologyMode) string {
	if topology == configv1.ExternalTopologyMode {
		return "Hypershift"
	}
	return "SelfManagedHA"
}

// collectEnabledGates returns the set of feature gate names currently enabled on the
// cluster. It uses the first (most recent) FeatureGateDetails entry in Status.FeatureGates.
func collectEnabledGates(fg *configv1.FeatureGate) []configv1.FeatureGateName {
	if len(fg.Status.FeatureGates) == 0 {
		return nil
	}
	// FeatureGates is keyed by payload version; index 0 is the most recently applied version.
	details := fg.Status.FeatureGates[0]
	gates := make([]configv1.FeatureGateName, 0, len(details.Enabled))
	for _, attr := range details.Enabled {
		gates = append(gates, attr.Name)
	}
	return gates
}

// kubeResourcesFromScheme derives the expected set of Kubernetes API resources at test time
// using the vendored client-go scheme and the DefaultAPIResourceConfigSource from k8s.io/kubernetes.
// This requires no manual list and stays correct across Kubernetes rebases automatically.
func kubeResourcesFromScheme() (sets.Set[schema.GroupVersionResource], error) {
	resourceConfig := controlplane.DefaultAPIResourceConfigSource()
	result := sets.New[schema.GroupVersionResource]()

	for gv, enabled := range resourceConfig.GroupVersionConfigs {
		if !enabled {
			continue
		}
		extra, err := resourcesForGroupVersion(gv)
		if err != nil {
			return nil, err
		}
		for gvr := range extra {
			result.Insert(gvr)
		}
	}

	// A small number of required Kubernetes resources live in separate vendored packages
	// (apiextensions-apiserver, kube-aggregator) and are not registered in clientgoscheme.
	// They are always served and hardcoded here to keep the no-manual-list guarantee for
	// everything in clientgoscheme while still covering these few outliers.
	for _, gvr := range kubeNonSchemeResources() {
		result.Insert(gvr)
	}
	return result, nil
}

// kubeNonSchemeResources lists Kubernetes built-in resources that are served by the
// kube-apiserver but whose types are not registered in k8s.io/client-go/kubernetes/scheme
// because they live in separate packages (apiextensions, kube-aggregator).
func kubeNonSchemeResources() []schema.GroupVersionResource {
	return []schema.GroupVersionResource{
		{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"},
		{Group: "apiregistration.k8s.io", Version: "v1", Resource: "apiservices"},
	}
}

// resourcesForGroupVersion returns the set of GVRs for a given GroupVersion using the
// client-go scheme and UnsafeGuessKindToResource. It filters out List, Options, and
// subresource-only types.
func resourcesForGroupVersion(gv schema.GroupVersion) (sets.Set[schema.GroupVersionResource], error) {
	result := sets.New[schema.GroupVersionResource]()
	types := clientgoscheme.Scheme.KnownTypes(gv)
	if len(types) == 0 {
		// GroupVersion is enabled in the config but has no types in the client-go scheme.
		// This is expected for some beta/alpha GVs that are enabled only via feature gates.
		return result, nil
	}
	for kind := range types {
		if shouldSkipKind(kind) {
			continue
		}
		plural, _ := meta.UnsafeGuessKindToResource(gv.WithKind(kind))
		result.Insert(plural)
	}
	return result, nil
}

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

// discoveredResources queries the cluster's API discovery and returns all served top-level
// resources (subresources containing "/" are excluded).
func discoveredResources(oc *exutil.CLI) (sets.Set[schema.GroupVersionResource], error) {
	discoveryClient := discovery.NewDiscoveryClientForConfigOrDie(oc.AdminConfig())
	_, resourceLists, err := discoveryClient.ServerGroupsAndResources()

	var groupFailed *discovery.ErrGroupDiscoveryFailed
	if err != nil && !errors.As(err, &groupFailed) {
		return nil, fmt.Errorf("discovery failed: %w", err)
	}
	if groupFailed != nil {
		// Log but continue — partial discovery is common during rolling restarts.
		for gv, gvErr := range groupFailed.Groups {
			framework.Logf("discovery failed for group %s: %v", gv, gvErr)
		}
	}

	result := sets.New[schema.GroupVersionResource]()
	for _, rl := range resourceLists {
		gv, err := schema.ParseGroupVersion(rl.GroupVersion)
		if err != nil {
			continue
		}
		for _, r := range rl.APIResources {
			if strings.Contains(r.Name, "/") {
				continue // skip subresources
			}
			result.Insert(gv.WithResource(r.Name))
		}
	}
	return result, nil
}

func gvrString(gvr schema.GroupVersionResource) string {
	if gvr.Group == "" {
		return fmt.Sprintf("core/%s/%s", gvr.Version, gvr.Resource)
	}
	return fmt.Sprintf("%s/%s/%s", gvr.Group, gvr.Version, gvr.Resource)
}

func formatGVRList(gvrs []schema.GroupVersionResource) string {
	strs := make([]string, 0, len(gvrs))
	for _, gvr := range gvrs {
		strs = append(strs, gvrString(gvr))
	}
	sort.Strings(strs)
	return strings.Join(strs, "\n")
}

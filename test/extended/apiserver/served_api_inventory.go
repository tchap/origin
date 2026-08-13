package apiserver

// E2E test for served API inventory validation.
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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/discovery"
	"k8s.io/kubernetes/test/e2e/framework"
	e2eskipper "k8s.io/kubernetes/test/e2e/framework/skipper"

	configv1 "github.com/openshift/api/config/v1"

	exutil "github.com/openshift/origin/test/extended/util"
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
		enabledGates := collectEnabledGates(featureGate)

		if featureGate.Spec.FeatureSet != configv1.Default {
			e2eskipper.Skipf("Test only runs on Default feature set, got %q", featureGate.Spec.FeatureSet)
		}

		// Get the Kubernetes version the cluster is actually running.
		serverVersion, err := oc.AdminKubeClient().Discovery().ServerVersion()
		o.Expect(err).NotTo(o.HaveOccurred())
		// ParseGeneric handles the "v" prefix and build metadata suffix (e.g. "+b9a44ad").
		kubeVersion, err := utilversion.ParseGeneric(serverVersion.GitVersion)
		o.Expect(err).NotTo(o.HaveOccurred())

		framework.Logf("cluster profile=%s featureSet=%q kubeVersion=%s enabledGates=%v",
			profile, featureGate.Spec.FeatureSet, kubeVersion, enabledGates)

		// 2. Build the expected API set.
		// forProfileAndVersion is a stub for servedapis.ForProfileAndVersion() in openshift/api.
		// It returns all expected APIs (OpenShift + Kubernetes) for the profile and kube version.
		// Returns found=false when data isn't available (unsupported profile, or during kube rebase).
		required, optional, found := forProfileAndVersion(profile, kubeVersion)
		if !found {
			e2eskipper.Skipf("API inventory for profile=%s kubeVersion=%d.%d not found. This is expected during Kubernetes rebase.", profile, kubeVersion.Major(), kubeVersion.Minor())
		}

		// 3. Apply Kubernetes API overrides for active OpenShift feature gates.
		// Some OpenShift feature gates enable alpha/beta Kubernetes APIs beyond the defaults.
		for _, gate := range enabledGates {
			for _, gvr := range kubeAPIOverridesForGate(gate, kubeVersion) {
				required = append(required, servedAPIEntry{
					Group:    gvr.Group,
					Version:  gvr.Version,
					Resource: gvr.Resource,
					Source:   sourceCoreKube,
				})
			}
		}

		// 4. Build expected GVR sets.
		requiredSet := sets.New[schema.GroupVersionResource]()
		optionalSet := sets.New[schema.GroupVersionResource]()
		for _, e := range required {
			requiredSet.Insert(schema.GroupVersionResource{Group: e.Group, Version: e.Version, Resource: e.Resource})
		}
		for _, e := range optional {
			optionalSet.Insert(schema.GroupVersionResource{Group: e.Group, Version: e.Version, Resource: e.Resource})
		}

		// 5. Query the cluster's actual served APIs via discovery.
		actual, err := discoveredResources(oc)
		o.Expect(err).NotTo(o.HaveOccurred())

		framework.Logf("expected required=%d optional=%d actual=%d", requiredSet.Len(), optionalSet.Len(), actual.Len())

		// Log optional API presence for visibility (not a failure either way).
		for _, gvr := range optionalSet.Intersection(actual).UnsortedList() {
			framework.Logf("optional API served: %s", gvrString(gvr))
		}
		for _, gvr := range optionalSet.Difference(actual).UnsortedList() {
			framework.Logf("optional API absent (OK): %s", gvrString(gvr))
		}

		// 6. Bidirectional comparison.
		missing := requiredSet.Difference(actual).UnsortedList()
		unexpected := actual.Difference(requiredSet.Union(optionalSet)).UnsortedList()

		o.Expect(missing).To(o.BeEmpty(),
			"required APIs not served by the cluster:\n%s", formatGVRList(missing))
		o.Expect(unexpected).To(o.BeEmpty(),
			"APIs served by the cluster but not in the expected list — add them to the servedapis stub or the optional list:\n%s", formatGVRList(unexpected))
	})
})

// clusterProfileName maps the infrastructure topology to the ClusterProfile used by the
// servedapis package (stub: clusterProfile; vendored: servedapis.ClusterProfile).
func clusterProfileName(topology configv1.TopologyMode) clusterProfile {
	if topology == configv1.ExternalTopologyMode {
		return clusterProfileHypershift
	}
	return clusterProfileSelfManagedHA
}

// collectEnabledGates returns the feature gate names currently enabled on the cluster.
// It uses the first (most recent) FeatureGateDetails entry in Status.FeatureGates.
func collectEnabledGates(fg *configv1.FeatureGate) []configv1.FeatureGateName {
	if len(fg.Status.FeatureGates) == 0 {
		return nil
	}
	details := fg.Status.FeatureGates[0]
	gates := make([]configv1.FeatureGateName, 0, len(details.Enabled))
	for _, attr := range details.Enabled {
		gates = append(gates, attr.Name)
	}
	return gates
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

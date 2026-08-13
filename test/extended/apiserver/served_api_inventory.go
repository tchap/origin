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
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/util/retry"
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

		if featureGate.Spec.FeatureSet != configv1.Default {
			e2eskipper.Skipf("Test only runs on Default feature set, got %q", featureGate.Spec.FeatureSet)
		}

		// Get the Kubernetes version the cluster is actually running.
		serverVersion, err := oc.AdminKubeClient().Discovery().ServerVersion()
		o.Expect(err).NotTo(o.HaveOccurred())
		// ParseGeneric handles the "v" prefix and build metadata suffix (e.g. "+b9a44ad").
		kubeVersion, err := utilversion.ParseGeneric(serverVersion.GitVersion)
		o.Expect(err).NotTo(o.HaveOccurred())

		framework.Logf("cluster profile=%s featureSet=%q kubeVersion=%s",
			profile, featureGate.Spec.FeatureSet, kubeVersion)

		// 2. Build the expected API set.
		// forProfileAndVersion is a stub for servedapis.ForProfileAndVersion() in openshift/api.
		// It returns all expected APIs (OpenShift + Kubernetes) for the Default feature set.
		// Returns found=false when data isn't available (unsupported profile, or during kube rebase).
		required, optional, found := forProfileAndVersion(profile, kubeVersion)
		if !found {
			e2eskipper.Skipf("API inventory for profile=%s kubeVersion=%d.%d not found. This is expected during Kubernetes rebase.", profile, kubeVersion.Major(), kubeVersion.Minor())
		}

		// 3. Build expected GVR sets.
		requiredSet := sets.New[schema.GroupVersionResource]()
		optionalSet := sets.New[schema.GroupVersionResource]()
		for _, e := range required {
			requiredSet.Insert(schema.GroupVersionResource{Group: e.Group, Version: e.Version, Resource: e.Resource})
		}
		for _, e := range optional {
			optionalSet.Insert(schema.GroupVersionResource{Group: e.Group, Version: e.Version, Resource: e.Resource})
		}

		// 4. Query the cluster's actual served APIs via discovery.
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

		// 5. Bidirectional comparison.
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

// discoveredResources queries the cluster's API discovery and returns all served top-level
// resources (subresources containing "/" are excluded). Retries on partial failures.
func discoveredResources(oc *exutil.CLI) (sets.Set[schema.GroupVersionResource], error) {
	discoveryClient := discovery.NewDiscoveryClientForConfigOrDie(oc.AdminConfig())
	var resourceLists []*metav1.APIResourceList

	// Retry on partial discovery failures (aggregated API servers restarting, etc.)
	// Worst case: 2s + 4s + 8s + (27 * 10s) ≈ 284s (~5 minutes).
	err := retry.OnError(
		wait.Backoff{
			Duration: 2 * time.Second,
			Steps:    30,
			Factor:   2.0,
			Jitter:   0.1,
			Cap:      10 * time.Second,
		},
		func(err error) bool {
			var groupFailed *discovery.ErrGroupDiscoveryFailed
			return errors.As(err, &groupFailed)
		},
		func() error {
			_, lists, err := discoveryClient.ServerGroupsAndResources()
			resourceLists = lists
			return err
		},
	)
	if err != nil {
		var groupFailed *discovery.ErrGroupDiscoveryFailed
		if errors.As(err, &groupFailed) {
			return nil, fmt.Errorf("discovery failed for %d groups after retries: %w", len(groupFailed.Groups), err)
		}
		return nil, fmt.Errorf("discovery failed: %w", err)
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

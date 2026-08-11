package apiserver

// Proof of concept for the served API inventory e2e test.
//
// The servedapis stubs in this file (apiSource, servedAPIEntry, servedAPIsForProfile,
// kubeAPIOverridesForGate) will be replaced by the vendored openshift/api servedapis
// and features packages once they exist. The function signatures intentionally match
// the planned API so the switchover is mechanical.

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"

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

// ---- Stubs for openshift/api servedapis package ----------------------------------------
// These types and functions stand in for the generated servedapis package that will live
// in openshift/api once Part A of the plan is implemented.

type apiSource string

const (
	sourceOpenShiftCRD       apiSource = "openshift-crd"
	sourceOpenShiftAPIServer apiSource = "openshift-apiserver"
	sourceOAuthAPIServer     apiSource = "oauth-apiserver"
	sourceOptional           apiSource = "optional"
)

type servedAPIEntry struct {
	Group    string
	Version  string
	Resource string
	Source   apiSource
}

// servedAPIsForProfile is a stub for servedapis.ForProfile(profile, featureSet).
// Returns a minimal hardcoded list of OpenShift API resources for the PoC.
// The real function is generated from CRD manifests and aggregated API server definitions.
func servedAPIsForProfile(profile, featureSet string) []servedAPIEntry {
	var result []servedAPIEntry
	result = append(result, openshiftAggregatedAPIs()...)
	result = append(result, openshiftCRDs(featureSet)...)
	result = append(result, openshiftOptionalAPIs()...)
	return result
}

func openshiftAggregatedAPIs() []servedAPIEntry {
	// openshift-apiserver resources (9 groups, all v1)
	osaResources := []struct{ group, resource string }{
		{"apps.openshift.io", "deploymentconfigs"},
		{"authorization.openshift.io", "clusterrolebindings"},
		{"authorization.openshift.io", "clusterroles"},
		{"authorization.openshift.io", "localresourceaccessreviews"},
		{"authorization.openshift.io", "localsubjectaccessreviews"},
		{"authorization.openshift.io", "resourceaccessreviews"},
		{"authorization.openshift.io", "rolebindingrestrictions"},
		{"authorization.openshift.io", "rolebindings"},
		{"authorization.openshift.io", "roles"},
		{"authorization.openshift.io", "selfsubjectrulesreviews"},
		{"authorization.openshift.io", "subjectaccessreviews"},
		{"authorization.openshift.io", "subjectrulesreviews"},
		{"build.openshift.io", "buildconfigs"},
		{"build.openshift.io", "builds"},
		{"image.openshift.io", "imagestreamimages"},
		{"image.openshift.io", "imagestreamimports"},
		{"image.openshift.io", "imagestreammappings"},
		{"image.openshift.io", "imagestreams"},
		{"image.openshift.io", "imagestreamtags"},
		{"image.openshift.io", "imagetags"},
		{"image.openshift.io", "images"},
		{"image.openshift.io", "imagesignatures"},
		{"project.openshift.io", "projectrequests"},
		{"project.openshift.io", "projects"},
		{"quota.openshift.io", "appliedclusterresourcequotas"},
		{"quota.openshift.io", "clusterresourcequotas"},
		{"route.openshift.io", "routes"},
		{"security.openshift.io", "podsecuritypolicyselfsubjectreviews"},
		{"security.openshift.io", "podsecuritypolicysubjectreviews"},
		{"security.openshift.io", "podsecuritypolicyreviews"},
		{"security.openshift.io", "rangeallocations"},
		{"security.openshift.io", "securitycontextconstraints"},
		{"template.openshift.io", "brokertemplateinstances"},
		{"template.openshift.io", "processedtemplates"},
		{"template.openshift.io", "templateinstances"},
		{"template.openshift.io", "templates"},
	}
	result := make([]servedAPIEntry, 0, len(osaResources))
	for _, r := range osaResources {
		result = append(result, servedAPIEntry{Group: r.group, Version: "v1", Resource: r.resource, Source: sourceOpenShiftAPIServer})
	}

	// oauth-apiserver resources (2 groups, all v1)
	oauthResources := []struct{ group, resource string }{
		{"oauth.openshift.io", "oauthaccesstokens"},
		{"oauth.openshift.io", "oauthauthorizetokens"},
		{"oauth.openshift.io", "oauthclientauthorizations"},
		{"oauth.openshift.io", "oauthclients"},
		{"oauth.openshift.io", "tokenreviews"},
		{"oauth.openshift.io", "useroauthaccesstokens"},
		{"user.openshift.io", "groups"},
		{"user.openshift.io", "identities"},
		{"user.openshift.io", "useridentitymappings"},
		{"user.openshift.io", "users"},
	}
	for _, r := range oauthResources {
		result = append(result, servedAPIEntry{Group: r.group, Version: "v1", Resource: r.resource, Source: sourceOAuthAPIServer})
	}
	return result
}

func openshiftCRDs(featureSet string) []servedAPIEntry {
	// OpenShift CRDs from payload-manifests/crds/. The real function reads all CRD manifests;
	// this stub covers the full set observed on a Default SelfManagedHA cluster.
	// techPreviewOnly=true means the CRD carries release.openshift.io/feature-set annotation
	// and is absent on Default clusters.
	type crdDef struct {
		group, version, resource string
		techPreviewOnly          bool
	}
	crds := []crdDef{
		// apiserver.openshift.io
		{"apiserver.openshift.io", "v1", "apirequestcounts", false},
		// cloud.network.openshift.io
		{"cloud.network.openshift.io", "v1", "cloudprivateipconfigs", false},
		// config.openshift.io
		{"config.openshift.io", "v1", "apiservers", false},
		{"config.openshift.io", "v1", "authentications", false},
		{"config.openshift.io", "v1", "builds", false},
		{"config.openshift.io", "v1", "clusterimagepolicies", false},
		{"config.openshift.io", "v1", "clusteroperators", false},
		{"config.openshift.io", "v1", "clusterversions", false},
		{"config.openshift.io", "v1", "consoles", false},
		{"config.openshift.io", "v1", "criocredentialproviderconfigs", false},
		{"config.openshift.io", "v1", "dnses", false},
		{"config.openshift.io", "v1", "featuregates", false},
		{"config.openshift.io", "v1", "imagecontentpolicies", false},
		{"config.openshift.io", "v1", "imagedigestmirrorsets", false},
		{"config.openshift.io", "v1", "imagepolicies", false},
		{"config.openshift.io", "v1", "images", false},
		{"config.openshift.io", "v1", "imagetagmirrorsets", false},
		{"config.openshift.io", "v1", "infrastructures", false},
		{"config.openshift.io", "v1", "ingresses", false},
		{"config.openshift.io", "v1", "insightsdatagathers", false},
		{"config.openshift.io", "v1", "networks", false},
		{"config.openshift.io", "v1", "nodes", false},
		{"config.openshift.io", "v1", "oauths", false},
		{"config.openshift.io", "v1", "operatorhubs", false},
		{"config.openshift.io", "v1", "projects", false},
		{"config.openshift.io", "v1", "proxies", false},
		{"config.openshift.io", "v1", "schedulers", false},
		// console.openshift.io
		{"console.openshift.io", "v1", "consoleclidownloads", false},
		{"console.openshift.io", "v1", "consoleexternalloglinks", false},
		{"console.openshift.io", "v1", "consolelinks", false},
		{"console.openshift.io", "v1", "consolenotifications", false},
		{"console.openshift.io", "v1", "consoleplugins", false},
		{"console.openshift.io", "v1", "consolequickstarts", false},
		{"console.openshift.io", "v1", "consolesamples", false},
		{"console.openshift.io", "v1", "consoleyamlsamples", false},
		// controlplane.operator.openshift.io
		{"controlplane.operator.openshift.io", "v1alpha1", "podnetworkconnectivitychecks", false},
		// imageregistry.operator.openshift.io
		{"imageregistry.operator.openshift.io", "v1", "configs", false},
		{"imageregistry.operator.openshift.io", "v1", "imagepruners", false},
		// ingress.operator.openshift.io
		{"ingress.operator.openshift.io", "v1", "dnsrecords", false},
		// insights.openshift.io
		{"insights.openshift.io", "v1", "datagathers", false},
		// machine.openshift.io
		{"machine.openshift.io", "v1", "controlplanemachinesets", false},
		// machineconfiguration.openshift.io
		{"machineconfiguration.openshift.io", "v1", "containerruntimeconfigs", false},
		{"machineconfiguration.openshift.io", "v1", "controllerconfigs", false},
		{"machineconfiguration.openshift.io", "v1", "kubeletconfigs", false},
		{"machineconfiguration.openshift.io", "v1", "machineconfignodes", false},
		{"machineconfiguration.openshift.io", "v1", "machineconfigpools", false},
		{"machineconfiguration.openshift.io", "v1", "machineconfigs", false},
		{"machineconfiguration.openshift.io", "v1", "machineosbuilds", false},
		{"machineconfiguration.openshift.io", "v1", "machineosconfigs", false},
		{"machineconfiguration.openshift.io", "v1", "pinnedimagesets", false},
		{"machineconfiguration.openshift.io", "v1", "internalreleaseimages", false},
		{"machineconfiguration.openshift.io", "v1", "osimagestreams", false},
		// monitoring.openshift.io
		{"monitoring.openshift.io", "v1", "alertingrules", false},
		{"monitoring.openshift.io", "v1", "alertrelabelconfigs", false},
		// network.operator.openshift.io
		{"network.operator.openshift.io", "v1", "egressrouters", false},
		{"network.operator.openshift.io", "v1", "operatorpkis", false},
		// operator.openshift.io
		{"operator.openshift.io", "v1", "authentications", false},
		{"operator.openshift.io", "v1", "cloudcredentials", false},
		{"operator.openshift.io", "v1", "clustercsidrivers", false},
		{"operator.openshift.io", "v1", "configs", false},
		{"operator.openshift.io", "v1", "consoles", false},
		{"operator.openshift.io", "v1", "csisnapshotcontrollers", false},
		{"operator.openshift.io", "v1", "dnses", false},
		{"operator.openshift.io", "v1", "etcds", false},
		{"operator.openshift.io", "v1", "ingresscontrollers", false},
		{"operator.openshift.io", "v1", "insightsoperators", false},
		{"operator.openshift.io", "v1", "kubeapiservers", false},
		{"operator.openshift.io", "v1", "kubecontrollermanagers", false},
		{"operator.openshift.io", "v1", "kubeschedulers", false},
		{"operator.openshift.io", "v1", "kubestorageversionmigrators", false},
		{"operator.openshift.io", "v1", "machineconfigurations", false},
		{"operator.openshift.io", "v1", "networks", false},
		{"operator.openshift.io", "v1", "olms", false},
		{"operator.openshift.io", "v1", "openshiftapiservers", false},
		{"operator.openshift.io", "v1", "openshiftcontrollermanagers", false},
		{"operator.openshift.io", "v1", "servicecas", false},
		{"operator.openshift.io", "v1", "storages", false},
		{"operator.openshift.io", "v1alpha1", "imagecontentsourcepolicies", false},
		// samples.operator.openshift.io
		{"samples.operator.openshift.io", "v1", "configs", false},
		// security.internal.openshift.io
		{"security.internal.openshift.io", "v1", "rangeallocations", false},
		// TechPreview-only CRDs
		{"config.openshift.io", "v1", "backups", true},
		{"config.openshift.io", "v1", "clustermonitorings", true},
		{"operator.openshift.io", "v1", "etcdbackups", true},
		{"config.openshift.io", "v1", "pkis", true},
	}
	isTechPreview := featureSet == string(configv1.TechPreviewNoUpgrade) ||
		featureSet == string(configv1.DevPreviewNoUpgrade)
	result := make([]servedAPIEntry, 0, len(crds))
	for _, c := range crds {
		if c.techPreviewOnly && !isTechPreview {
			continue
		}
		result = append(result, servedAPIEntry{Group: c.group, Version: c.version, Resource: c.resource, Source: sourceOpenShiftCRD})
	}
	return result
}

func openshiftOptionalAPIs() []servedAPIEntry {
	// APIs from optional operators/components outside the core payload, or from third-party
	// operators commonly installed on OpenShift clusters. Their absence is not a failure.
	optional := []struct{ group, version, resource string }{
		// Cluster Autoscaler / Machine Autoscaler
		{"autoscaling.openshift.io", "v1", "clusterautoscalers"},
		{"autoscaling.openshift.io", "v1beta1", "machineautoscalers"},
		// Machine API (optional on some profiles)
		{"machine.openshift.io", "v1beta1", "machinehealthchecks"},
		{"machine.openshift.io", "v1beta1", "machines"},
		{"machine.openshift.io", "v1beta1", "machinesets"},
		// Cloud Credential Operator
		{"cloudcredential.openshift.io", "v1", "credentialsrequests"},
		// Helm
		{"helm.openshift.io", "v1beta1", "helmchartrepositories"},
		{"helm.openshift.io", "v1beta1", "projecthelmchartrepositories"},
		// Prometheus / monitoring stack (prometheus-operator)
		{"monitoring.coreos.com", "v1", "alertmanagers"},
		{"monitoring.coreos.com", "v1", "podmonitors"},
		{"monitoring.coreos.com", "v1", "probes"},
		{"monitoring.coreos.com", "v1", "prometheuses"},
		{"monitoring.coreos.com", "v1", "prometheusrules"},
		{"monitoring.coreos.com", "v1", "servicemonitors"},
		{"monitoring.coreos.com", "v1", "thanosrulers"},
		{"monitoring.coreos.com", "v1alpha1", "alertmanagerconfigs"},
		{"monitoring.coreos.com", "v1beta1", "alertmanagerconfigs"},
		// OLM v1 (classic)
		{"operators.coreos.com", "v1", "olmconfigs"},
		{"operators.coreos.com", "v1", "operatorconditions"},
		{"operators.coreos.com", "v1", "operatorgroups"},
		{"operators.coreos.com", "v1", "operators"},
		{"operators.coreos.com", "v1alpha1", "catalogsources"},
		{"operators.coreos.com", "v1alpha1", "clusterserviceversions"},
		{"operators.coreos.com", "v1alpha1", "installplans"},
		{"operators.coreos.com", "v1alpha1", "subscriptions"},
		{"operators.coreos.com", "v1alpha2", "operatorgroups"},
		{"operators.coreos.com", "v2", "operatorconditions"},
		{"packages.operators.coreos.com", "v1", "packagemanifests"},
		// OLM v1 (new)
		{"olm.operatorframework.io", "v1", "clustercatalogs"},
		{"olm.operatorframework.io", "v1", "clusterextensions"},
		// Metal3 / baremetal
		{"metal3.io", "v1alpha1", "baremetalhosts"},
		{"metal3.io", "v1alpha1", "bmceventsubscriptions"},
		{"metal3.io", "v1alpha1", "dataimages"},
		{"metal3.io", "v1alpha1", "firmwareschemas"},
		{"metal3.io", "v1alpha1", "hardwaredata"},
		{"metal3.io", "v1alpha1", "hostfirmwarecomponents"},
		{"metal3.io", "v1alpha1", "hostfirmwaresettings"},
		{"metal3.io", "v1alpha1", "hostupdatepolicies"},
		{"metal3.io", "v1alpha1", "preprovisioningimages"},
		{"metal3.io", "v1alpha1", "provisionings"},
		{"infrastructure.cluster.x-k8s.io", "v1beta1", "metal3remediations"},
		{"infrastructure.cluster.x-k8s.io", "v1beta1", "metal3remediationtemplates"},
		// IPAM (Cluster API)
		{"ipam.cluster.x-k8s.io", "v1alpha1", "ipaddressclaims"},
		{"ipam.cluster.x-k8s.io", "v1alpha1", "ipaddresses"},
		{"ipam.cluster.x-k8s.io", "v1beta1", "ipaddressclaims"},
		{"ipam.cluster.x-k8s.io", "v1beta1", "ipaddresses"},
		// Node Tuning Operator
		{"tuned.openshift.io", "v1", "profiles"},
		{"tuned.openshift.io", "v1", "tuneds"},
		// Performance Addon Operator
		{"performance.openshift.io", "v1", "performanceprofiles"},
		{"performance.openshift.io", "v1alpha1", "performanceprofiles"},
		{"performance.openshift.io", "v2", "performanceprofiles"},
		// Multus / CNI
		{"k8s.cni.cncf.io", "v1", "network-attachment-definitions"},
		{"k8s.cni.cncf.io", "v1alpha1", "ipamclaims"},
		{"whereabouts.cni.cncf.io", "v1alpha1", "ippools"},
		{"whereabouts.cni.cncf.io", "v1alpha1", "nodeslicepools"},
		{"whereabouts.cni.cncf.io", "v1alpha1", "overlappingrangeipreservations"},
		// OVN-Kubernetes
		{"k8s.ovn.org", "v1", "adminpolicybasedexternalroutes"},
		{"k8s.ovn.org", "v1", "clusteruserdefinednetworks"},
		{"k8s.ovn.org", "v1", "egressfirewalls"},
		{"k8s.ovn.org", "v1", "egressips"},
		{"k8s.ovn.org", "v1", "egressqoses"},
		{"k8s.ovn.org", "v1", "egressservices"},
		{"k8s.ovn.org", "v1", "userdefinednetworks"},
		// Gateway API (installed via GatewayAPIWithoutOLM feature gate or manually)
		{"gateway.networking.k8s.io", "v1", "backendtlspolicies"},
		{"gateway.networking.k8s.io", "v1", "gatewayclasses"},
		{"gateway.networking.k8s.io", "v1", "gateways"},
		{"gateway.networking.k8s.io", "v1", "grpcroutes"},
		{"gateway.networking.k8s.io", "v1", "httproutes"},
		{"gateway.networking.k8s.io", "v1", "listenersets"},
		{"gateway.networking.k8s.io", "v1", "referencegrants"},
		{"gateway.networking.k8s.io", "v1", "tlsroutes"},
		{"gateway.networking.k8s.io", "v1beta1", "gatewayclasses"},
		{"gateway.networking.k8s.io", "v1beta1", "gateways"},
		{"gateway.networking.k8s.io", "v1beta1", "httproutes"},
		{"gateway.networking.k8s.io", "v1beta1", "referencegrants"},
		// Admin Network Policy
		{"policy.networking.k8s.io", "v1alpha1", "adminnetworkpolicies"},
		{"policy.networking.k8s.io", "v1alpha1", "baselineadminnetworkpolicies"},
		// Volume snapshot / populator
		{"snapshot.storage.k8s.io", "v1", "volumesnapshotclasses"},
		{"snapshot.storage.k8s.io", "v1", "volumesnapshotcontents"},
		{"snapshot.storage.k8s.io", "v1", "volumesnapshots"},
		{"populator.storage.k8s.io", "v1beta1", "volumepopulators"},
		// Metrics server (aggregated API)
		{"metrics.k8s.io", "v1beta1", "nodes"},
		{"metrics.k8s.io", "v1beta1", "pods"},
		// Storage migration
		{"migration.k8s.io", "v1alpha1", "storagestates"},
		{"migration.k8s.io", "v1alpha1", "storageversionmigrations"},
		// OpenShift Tests Extension admission CRD (CI infrastructure)
		{"testextension.redhat.io", "v1", "testextensionadmissions"},
	}
	result := make([]servedAPIEntry, 0, len(optional))
	for _, r := range optional {
		result = append(result, servedAPIEntry{Group: r.group, Version: r.version, Resource: r.resource, Source: sourceOptional})
	}
	return result
}

// kubeAPIOverridesForGate is a stub for the openshift/api feature gate → additional k8s
// GVRs mapping. The real data lives in cluster-kube-apiserver-operator; the plan moves it
// to openshift/api so origin can vendor it.
//
// Returns specific GVRs (not whole GroupVersions) to avoid pulling in stale scheme
// registrations for types that graduated to v1 but remain in the scheme for serialisation
// backwards-compat (e.g. MutatingWebhookConfiguration at v1beta1).
func kubeAPIOverridesForGate(gate configv1.FeatureGateName) []schema.GroupVersionResource {
	overrides := map[configv1.FeatureGateName][]schema.GroupVersionResource{
		"MutatingAdmissionPolicy": {
			{Group: "admissionregistration.k8s.io", Version: "v1alpha1", Resource: "mutatingadmissionpolicies"},
			{Group: "admissionregistration.k8s.io", Version: "v1alpha1", Resource: "mutatingadmissionpolicybindings"},
			{Group: "admissionregistration.k8s.io", Version: "v1beta1", Resource: "mutatingadmissionpolicies"},
			{Group: "admissionregistration.k8s.io", Version: "v1beta1", Resource: "mutatingadmissionpolicybindings"},
		},
		"ValidatingAdmissionPolicy": {
			// Graduated to v1 in Kubernetes 1.30; present here only for clusters still on v1beta1.
			{Group: "admissionregistration.k8s.io", Version: "v1beta1", Resource: "validatingadmissionpolicies"},
			{Group: "admissionregistration.k8s.io", Version: "v1beta1", Resource: "validatingadmissionpolicybindings"},
		},
	}
	return overrides[gate]
}

// ---- End of stubs -----------------------------------------------------------------------

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

		featureGate, err := oc.AdminConfigClient().ConfigV1().FeatureGates().Get(ctx, "cluster", metav1.GetOptions{})
		o.Expect(err).NotTo(o.HaveOccurred())

		profile := clusterProfileName(*topology)
		featureSet := string(featureGate.Spec.FeatureSet)
		enabledGates := collectEnabledGates(featureGate)

		framework.Logf("cluster profile=%s featureSet=%q enabledGates=%v", profile, featureSet, enabledGates)

		// 2. Build the OpenShift expected API set (stub; will use vendored servedapis package).
		openshiftEntries := servedAPIsForProfile(profile, featureSet)

		// 3. Build the Kubernetes stable expected API set from the vendored scheme.
		expectedKube, err := kubeResourcesFromScheme()
		o.Expect(err).NotTo(o.HaveOccurred())

		// 4. Apply Kubernetes overrides from active OpenShift feature gates.
		for _, gate := range enabledGates {
			for _, gvr := range kubeAPIOverridesForGate(gate) {
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

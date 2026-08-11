package apiserver

// Stubs for the openshift/api servedapis and features packages.
//
// These stand in for the generated code that will live in openshift/api once Part A of
// the plan is implemented. When that package is vendored into origin:
//   1. Delete this file.
//   2. Add:  import "github.com/openshift/api/servedapis"
//            import "github.com/openshift/api/features"
//   3. Replace servedAPIsForProfile → servedapis.ForProfile
//            kubeAPIOverridesForGate → features.KubeAPIOverridesForGate
//            servedAPIEntry           → servedapis.ServedAPIEntry
//            apiSource / sourceXxx   → servedapis.Source / servedapis.SourceXxx

import (
	"github.com/blang/semver/v4"

	"k8s.io/apimachinery/pkg/runtime/schema"

	configv1 "github.com/openshift/api/config/v1"
)

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

// servedAPIsForProfile is a stub for servedapis.ForProfile(clusterProfile, featureSet).
// Returns a hardcoded list of OpenShift API resources for the PoC.
// The real function is generated from CRD manifests and aggregated API server definitions.
func servedAPIsForProfile(clusterProfile, featureSet string) []servedAPIEntry {
	var result []servedAPIEntry
	result = append(result, openshiftAggregatedAPIs(clusterProfile)...)
	result = append(result, openshiftCRDs(clusterProfile, featureSet)...)
	result = append(result, openshiftOptionalAPIs()...)
	return result
}

func openshiftAggregatedAPIs(clusterProfile string) []servedAPIEntry {
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

	// oauth-apiserver resources (2 groups, all v1).
	// On HyperShift clusters with external OIDC, oauth-apiserver may not be deployed,
	// so these are only required for the SelfManagedHA profile.
	if clusterProfile == "SelfManagedHA" {
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
	}
	return result
}

func openshiftCRDs(clusterProfile, featureSet string) []servedAPIEntry {
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

	// Machine API is required on SelfManagedHA clusters but absent on HyperShift
	// (worker nodes on HyperShift are managed by the management cluster, not the hosted cluster).
	if clusterProfile == "SelfManagedHA" {
		for _, r := range []struct{ version, resource string }{
			{"v1beta1", "machinehealthchecks"},
			{"v1beta1", "machines"},
			{"v1beta1", "machinesets"},
		} {
			result = append(result, servedAPIEntry{Group: "machine.openshift.io", Version: r.version, Resource: r.resource, Source: sourceOpenShiftCRD})
		}
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

// kubeAPIOverride is a stub for features.KubeAPIOverride in openshift/api.
// Mirrors the structure from cluster-kube-apiserver-operator's groupVersionByOpenshiftVersion,
// extended to GVR granularity. The real openshift/api implementation uses GroupVersion +
// scheme-based derivation; the stub uses explicit GVRs because the client-go scheme retains
// historical registrations (e.g. ValidatingAdmissionPolicy at v1alpha1, webhook configs at
// v1beta1) for types that have since graduated, making scheme-based derivation inaccurate
// for alpha/beta override GVs.
type kubeAPIOverride struct {
	schema.GroupVersionResource
	// KubeVersionRange filters which Kubernetes versions this entry applies to.
	// nil means all versions.
	KubeVersionRange semver.Range
}

// defaultGVRsByFeatureGate is a stub for features.KubeAPIOverridesByFeatureGate in openshift/api.
// Ported from cluster-kube-apiserver-operator's defaultGroupVersionsByFeatureGate, with
// GVRs replacing GVs to avoid pulling in stale scheme registrations.
var defaultGVRsByFeatureGate = map[configv1.FeatureGateName][]kubeAPIOverride{
	"MutatingAdmissionPolicy": {
		// Both v1alpha1 and v1beta1 must be served pre-GA because e2e tests exercise both.
		// TODO: Remove once openshift-apiserver is rebased to k8s 1.36+ (MutatingAdmissionPolicy v1).
		{KubeVersionRange: semver.MustParseRange(">=1.33.0 <1.37.0"), GroupVersionResource: schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1alpha1", Resource: "mutatingadmissionpolicies"}},
		{KubeVersionRange: semver.MustParseRange(">=1.33.0 <1.37.0"), GroupVersionResource: schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1alpha1", Resource: "mutatingadmissionpolicybindings"}},
		{KubeVersionRange: semver.MustParseRange(">=1.34.0 <1.37.0"), GroupVersionResource: schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1beta1", Resource: "mutatingadmissionpolicies"}},
		{KubeVersionRange: semver.MustParseRange(">=1.34.0 <1.37.0"), GroupVersionResource: schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1beta1", Resource: "mutatingadmissionpolicybindings"}},
	},
}

// kubeAPIOverridesByVersion filters defaultGVRsByFeatureGate to only the entries that apply
// to the given Kubernetes version, returning the active GVRs per gate.
// Stub for features.KubeAPIOverridesByVersion in openshift/api.
func kubeAPIOverridesByVersion(kubeVersion semver.Version) map[configv1.FeatureGateName][]schema.GroupVersionResource {
	result := make(map[configv1.FeatureGateName][]schema.GroupVersionResource, len(defaultGVRsByFeatureGate))
	for gate, entries := range defaultGVRsByFeatureGate {
		for _, e := range entries {
			if e.KubeVersionRange == nil || e.KubeVersionRange(kubeVersion) {
				result[gate] = append(result[gate], e.GroupVersionResource)
			}
		}
	}
	return result
}

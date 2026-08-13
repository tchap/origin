package apiserver

// Stubs for the openshift/api servedapis package.
//
// These stand in for the generated code that will live in openshift/api once Part A of
// the plan is implemented. When that package is vendored into origin:
//   1. Delete this file.
//   2. Add:  import "github.com/openshift/api/servedapis"
//   3. Replace:
//      clusterProfile       → servedapis.ClusterProfile
//      clusterProfileXxx    → servedapis.ClusterProfileXxx
//      source / sourceXxx   → servedapis.Source / servedapis.SourceXxx
//      servedAPIEntry       → servedapis.ServedAPIEntry
//      forProfileAndVersion → servedapis.ForProfileAndVersion

import (
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/kubernetes/pkg/controlplane"
)

type clusterProfile string

const (
	clusterProfileSelfManagedHA clusterProfile = "SelfManagedHA"
	clusterProfileHypershift    clusterProfile = "Hypershift"
)

type source string

const (
	sourceCoreKube           source = "core-kube"
	sourceOpenShiftCRD       source = "openshift-crd"
	sourceOpenShiftAPIServer source = "openshift-apiserver"
	sourceOAuthAPIServer     source = "oauth-apiserver"
)

type servedAPIEntry struct {
	Group    string
	Version  string
	Resource string
	Source   source
}

// forProfileAndVersion is a stub for servedapis.ForProfileAndVersion in openshift/api.
// Returns required and optional API entries for the Default feature set at the given
// cluster profile and Kubernetes version. Includes APIs enabled by Default feature set gates.
// Returns found=false when data is unavailable (unsupported profile, or during kube rebase).
func forProfileAndVersion(profile clusterProfile, kubeVersion *utilversion.Version) (required, optional []servedAPIEntry, found bool) {
	if profile != clusterProfileSelfManagedHA {
		// Hypershift stub data is not yet generated; the test will skip.
		return nil, nil, false
	}

	var req []servedAPIEntry
	req = append(req, openshiftAggregatedAPIs(profile)...)
	req = append(req, openshiftCRDs(profile)...)

	// Kubernetes APIs: derived from scheme + DefaultAPIResourceConfigSource at test time.
	// In the real openshift/api implementation these are pre-generated per kube version.
	kubeEntries, err := kubeResourcesFromScheme()
	if err != nil {
		return nil, nil, false
	}
	for gvr := range kubeEntries {
		req = append(req, servedAPIEntry{
			Group:    gvr.Group,
			Version:  gvr.Version,
			Resource: gvr.Resource,
			Source:   sourceCoreKube,
		})
	}

	return req, openshiftOptionalAPIs(), true
}

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

// kubeResourcesFromScheme derives the expected set of Kubernetes API resources using
// the vendored client-go scheme and DefaultAPIResourceConfigSource.
// In the real openshift/api implementation this is pre-generated per kube version; here
// it runs at test time against the currently vendored scheme.
func kubeResourcesFromScheme() (sets.Set[schema.GroupVersionResource], error) {
	resourceConfig := controlplane.DefaultAPIResourceConfigSource()
	result := sets.New[schema.GroupVersionResource]()

	for gv, enabled := range resourceConfig.GroupVersionConfigs {
		if !enabled {
			continue
		}
		for kind := range clientgoscheme.Scheme.KnownTypes(gv) {
			if shouldSkipKind(kind) {
				continue
			}
			plural, _ := meta.UnsafeGuessKindToResource(gv.WithKind(kind))
			result.Insert(plural)
		}
	}

	// A small number of required Kubernetes resources live in separate vendored packages
	// (apiextensions-apiserver, kube-aggregator) and are not registered in clientgoscheme.
	for _, gvr := range []schema.GroupVersionResource{
		{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"},
		{Group: "apiregistration.k8s.io", Version: "v1", Resource: "apiservices"},
	} {
		result.Insert(gvr)
	}
	return result, nil
}

func openshiftAggregatedAPIs(profile clusterProfile) []servedAPIEntry {
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
	if profile == clusterProfileSelfManagedHA {
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

func openshiftCRDs(profile clusterProfile) []servedAPIEntry {
	// OpenShift CRDs from payload-manifests/crds/. The real implementation reads all CRD manifests;
	// this stub covers the full set observed on a Default SelfManagedHA cluster.
	// TechPreview-only CRDs are excluded — the test skips on non-Default feature sets.
	crds := []struct{ group, version, resource string }{
		// apiserver.openshift.io
		{"apiserver.openshift.io", "v1", "apirequestcounts"},
		// cloud.network.openshift.io
		{"cloud.network.openshift.io", "v1", "cloudprivateipconfigs"},
		// config.openshift.io
		{"config.openshift.io", "v1", "apiservers"},
		{"config.openshift.io", "v1", "authentications"},
		{"config.openshift.io", "v1", "builds"},
		{"config.openshift.io", "v1", "clusterimagepolicies"},
		{"config.openshift.io", "v1", "clusteroperators"},
		{"config.openshift.io", "v1", "clusterversions"},
		{"config.openshift.io", "v1", "consoles"},
		{"config.openshift.io", "v1", "criocredentialproviderconfigs"},
		{"config.openshift.io", "v1", "dnses"},
		{"config.openshift.io", "v1", "featuregates"},
		{"config.openshift.io", "v1", "imagecontentpolicies"},
		{"config.openshift.io", "v1", "imagedigestmirrorsets"},
		{"config.openshift.io", "v1", "imagepolicies"},
		{"config.openshift.io", "v1", "images"},
		{"config.openshift.io", "v1", "imagetagmirrorsets"},
		{"config.openshift.io", "v1", "infrastructures"},
		{"config.openshift.io", "v1", "ingresses"},
		{"config.openshift.io", "v1", "insightsdatagathers"},
		{"config.openshift.io", "v1", "networks"},
		{"config.openshift.io", "v1", "nodes"},
		{"config.openshift.io", "v1", "oauths"},
		{"config.openshift.io", "v1", "operatorhubs"},
		{"config.openshift.io", "v1", "projects"},
		{"config.openshift.io", "v1", "proxies"},
		{"config.openshift.io", "v1", "schedulers"},
		// console.openshift.io
		{"console.openshift.io", "v1", "consoleclidownloads"},
		{"console.openshift.io", "v1", "consoleexternalloglinks"},
		{"console.openshift.io", "v1", "consolelinks"},
		{"console.openshift.io", "v1", "consolenotifications"},
		{"console.openshift.io", "v1", "consoleplugins"},
		{"console.openshift.io", "v1", "consolequickstarts"},
		{"console.openshift.io", "v1", "consolesamples"},
		{"console.openshift.io", "v1", "consoleyamlsamples"},
		// controlplane.operator.openshift.io
		{"controlplane.operator.openshift.io", "v1alpha1", "podnetworkconnectivitychecks"},
		// imageregistry.operator.openshift.io
		{"imageregistry.operator.openshift.io", "v1", "configs"},
		{"imageregistry.operator.openshift.io", "v1", "imagepruners"},
		// ingress.operator.openshift.io
		{"ingress.operator.openshift.io", "v1", "dnsrecords"},
		// insights.openshift.io
		{"insights.openshift.io", "v1", "datagathers"},
		// machine.openshift.io
		{"machine.openshift.io", "v1", "controlplanemachinesets"},
		// machineconfiguration.openshift.io
		{"machineconfiguration.openshift.io", "v1", "containerruntimeconfigs"},
		{"machineconfiguration.openshift.io", "v1", "controllerconfigs"},
		{"machineconfiguration.openshift.io", "v1", "kubeletconfigs"},
		{"machineconfiguration.openshift.io", "v1", "machineconfignodes"},
		{"machineconfiguration.openshift.io", "v1", "machineconfigpools"},
		{"machineconfiguration.openshift.io", "v1", "machineconfigs"},
		{"machineconfiguration.openshift.io", "v1", "machineosbuilds"},
		{"machineconfiguration.openshift.io", "v1", "machineosconfigs"},
		{"machineconfiguration.openshift.io", "v1", "pinnedimagesets"},
		{"machineconfiguration.openshift.io", "v1", "internalreleaseimages"},
		{"machineconfiguration.openshift.io", "v1", "osimagestreams"},
		// monitoring.openshift.io
		{"monitoring.openshift.io", "v1", "alertingrules"},
		{"monitoring.openshift.io", "v1", "alertrelabelconfigs"},
		// network.operator.openshift.io
		{"network.operator.openshift.io", "v1", "egressrouters"},
		{"network.operator.openshift.io", "v1", "operatorpkis"},
		// operator.openshift.io
		{"operator.openshift.io", "v1", "authentications"},
		{"operator.openshift.io", "v1", "cloudcredentials"},
		{"operator.openshift.io", "v1", "clustercsidrivers"},
		{"operator.openshift.io", "v1", "configs"},
		{"operator.openshift.io", "v1", "consoles"},
		{"operator.openshift.io", "v1", "csisnapshotcontrollers"},
		{"operator.openshift.io", "v1", "dnses"},
		{"operator.openshift.io", "v1", "etcds"},
		{"operator.openshift.io", "v1", "ingresscontrollers"},
		{"operator.openshift.io", "v1", "insightsoperators"},
		{"operator.openshift.io", "v1", "kubeapiservers"},
		{"operator.openshift.io", "v1", "kubecontrollermanagers"},
		{"operator.openshift.io", "v1", "kubeschedulers"},
		{"operator.openshift.io", "v1", "kubestorageversionmigrators"},
		{"operator.openshift.io", "v1", "machineconfigurations"},
		{"operator.openshift.io", "v1", "networks"},
		{"operator.openshift.io", "v1", "olms"},
		{"operator.openshift.io", "v1", "openshiftapiservers"},
		{"operator.openshift.io", "v1", "openshiftcontrollermanagers"},
		{"operator.openshift.io", "v1", "servicecas"},
		{"operator.openshift.io", "v1", "storages"},
		{"operator.openshift.io", "v1alpha1", "imagecontentsourcepolicies"},
		// samples.operator.openshift.io
		{"samples.operator.openshift.io", "v1", "configs"},
		// security.internal.openshift.io
		{"security.internal.openshift.io", "v1", "rangeallocations"},
	}

	result := make([]servedAPIEntry, 0, len(crds))
	for _, c := range crds {
		result = append(result, servedAPIEntry{Group: c.group, Version: c.version, Resource: c.resource, Source: sourceOpenShiftCRD})
	}

	// Machine API is required on SelfManagedHA clusters but absent on HyperShift
	// (worker nodes on HyperShift are managed by the management cluster).
	if profile == clusterProfileSelfManagedHA {
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
	// APIs from optional operators/components outside the core payload. Their absence is not a failure.
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
		result = append(result, servedAPIEntry{Group: r.group, Version: r.version, Resource: r.resource, Source: sourceOpenShiftCRD})
	}
	return result
}

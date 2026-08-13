package apiserver

import (
	"strings"

	"github.com/blang/semver/v4"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/openshift/origin/test/extended/apiserver/inventory"
)

// Stubs for the openshift/api servedapis package.
//
// These stand in for the generated code that will live in openshift/api once that's implemented.
// This file contains ONLY OpenShift APIs (CRDs, aggregated servers, optional operators).
// Kubernetes APIs are now generated in test/extended/apiserver/inventory/zz_generated_kubernetes.go.
//
// When openshift/api servedapis package is vendored:
//   1. Delete this file.
//   2. Add:  import "github.com/openshift/api/servedapis"
//   3. Replace stub types with vendored types in served_api_inventory.go

type clusterProfile string

const (
	clusterProfileSelfManagedHA clusterProfile = "SelfManagedHA"
	clusterProfileHypershift    clusterProfile = "Hypershift"
)

type servedAPIEntry struct {
	Group    string
	Version  string
	Resource string
}

// forProfile is a stub for servedapis.ForProfile in openshift/api.
// Returns required and optional OpenShift API entries for the Default feature set.
// Kubernetes APIs are returned separately from inventory.ForKubeVersion().
// Returns found=false when data is unavailable (unsupported profile).
func forProfile(profile clusterProfile) (required, optional []servedAPIEntry, found bool) {
	if profile != clusterProfileSelfManagedHA {
		// Hypershift stub data is not yet generated; the test will skip.
		return nil, nil, false
	}

	var req []servedAPIEntry
	req = append(req, openshiftAggregatedAPIs(profile)...)
	req = append(req, openshiftCRDs(profile)...)

	return req, openshiftOptionalAPIs(), true
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
		result = append(result, servedAPIEntry{Group: r.group, Version: "v1", Resource: r.resource, })
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
			result = append(result, servedAPIEntry{Group: r.group, Version: "v1", Resource: r.resource, })
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
		result = append(result, servedAPIEntry{Group: c.group, Version: c.version, Resource: c.resource, })
	}

	// Machine API is required on SelfManagedHA clusters but absent on HyperShift
	// (worker nodes on HyperShift are managed by the management cluster).
	if profile == clusterProfileSelfManagedHA {
		for _, r := range []struct{ version, resource string }{
			{"v1beta1", "machinehealthchecks"},
			{"v1beta1", "machines"},
			{"v1beta1", "machinesets"},
		} {
			result = append(result, servedAPIEntry{Group: "machine.openshift.io", Version: r.version, Resource: r.resource, })
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
		result = append(result, servedAPIEntry{Group: r.group, Version: r.version, Resource: r.resource, })
	}
	return result
}

// Kubernetes API overrides - stub for openshift/api features.KubeAPIOverridesByFeatureGate

type kubeAPIOverride struct {
	GroupVersion     string
	Kinds            []string
	KubeVersionRange semver.Range // nil means all versions
}

// kubeAPIOverridesByFeatureGate maps OpenShift feature gate names to additional Kubernetes
// APIs they enable beyond upstream DefaultAPIResourceConfigSource.
// This is a stub - the real implementation will come from openshift/api features package.
var kubeAPIOverridesByFeatureGate = map[string][]kubeAPIOverride{
	"MutatingAdmissionPolicy": {
		{
			GroupVersion:     "admissionregistration.k8s.io/v1alpha1",
			Kinds:            []string{"MutatingAdmissionPolicy", "MutatingAdmissionPolicyBinding"},
			KubeVersionRange: semver.MustParseRange(">=1.33.0 <1.34.0"),
		},
		{
			GroupVersion:     "admissionregistration.k8s.io/v1beta1",
			Kinds:            []string{"MutatingAdmissionPolicy", "MutatingAdmissionPolicyBinding"},
			KubeVersionRange: semver.MustParseRange(">=1.34.0 <1.37.0"),
		},
	},
	"VolumeGroupSnapshot": {
		{
			GroupVersion:     "groupsnapshot.storage.k8s.io/v1",
			Kinds:            []string{"VolumeGroupSnapshot", "VolumeGroupSnapshotClass", "VolumeGroupSnapshotContent"},
			KubeVersionRange: nil, // all versions
		},
	},
}

// applyFeatureGateOverrides adds Kubernetes APIs enabled by OpenShift feature gates
// to the base inventory. This is a stub - will use openshift/api features package later.
func applyFeatureGateOverrides(baseAPIs []inventory.ServedAPIEntry, enabledGates map[string]bool, kubeVersionStr string) []servedAPIEntry {
	// Parse version for range matching
	semVersion, err := semver.ParseTolerant(kubeVersionStr)
	if err != nil {
		// If parsing fails, just convert base APIs without overrides
		return convertToStubFormat(baseAPIs)
	}

	result := make([]servedAPIEntry, 0, len(baseAPIs)+10)

	// Convert base APIs
	for _, e := range baseAPIs {
		result = append(result, servedAPIEntry{
			Group:    e.Group,
			Version:  e.Version,
			Resource: e.Resource,
		})
	}

	// Add APIs for each enabled feature gate
	for gateName, enabled := range enabledGates {
		if !enabled {
			continue
		}

		overrides, found := kubeAPIOverridesByFeatureGate[gateName]
		if !found {
			continue
		}

		for _, override := range overrides {
			// Skip if version range doesn't match
			if override.KubeVersionRange != nil && !override.KubeVersionRange(semVersion) {
				continue
			}

			// Parse GroupVersion
			parts := strings.SplitN(override.GroupVersion, "/", 2)
			if len(parts) != 2 {
				continue
			}
			group, version := parts[0], parts[1]

			// Add each Kind from this override
			for _, kind := range override.Kinds {
				// Use proper pluralization via meta.UnsafeGuessKindToResource
				gvk := schema.GroupVersionKind{
					Group:   group,
					Version: version,
					Kind:    kind,
				}
				gvr, _ := meta.UnsafeGuessKindToResource(gvk)
				result = append(result, servedAPIEntry{
					Group:    gvr.Group,
					Version:  gvr.Version,
					Resource: gvr.Resource,
				})
			}
		}
	}

	return result
}

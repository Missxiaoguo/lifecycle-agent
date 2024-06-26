/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package extramanifest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	sriovv1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	ibuv1 "github.com/openshift-kni/lifecycle-agent/api/imagebasedupgrade/v1"
	"github.com/openshift-kni/lifecycle-agent/internal/common"
	"github.com/openshift-kni/lifecycle-agent/utils"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/scheme"
	policiesv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	testscheme = scheme.Scheme
)

func init() {
	testscheme.AddKnownTypes(policiesv1.GroupVersion, &policiesv1.Policy{})
	testscheme.AddKnownTypes(policiesv1.GroupVersion, &policiesv1.PolicyList{})
	testscheme.AddKnownTypes(apiextensionsv1.SchemeGroupVersion, &apiextensionsv1.CustomResourceDefinition{})
	testscheme.AddKnownTypes(ibuv1.SchemeGroupVersion, &ibuv1.ImageBasedUpgrade{})
}

const sriovnodepolicies = `
apiVersion: sriovnetwork.openshift.io/v1
kind: SriovNetworkNodePolicy
metadata:
  name: sriov-nnp-mh
  namespace: openshift-sriov-network-operator
  annotations:
    lca.openshift.io/apply-wave: "1"   
  spec:
    deviceType: netdevice
    isRdma: false
---
apiVersion: sriovnetwork.openshift.io/v1
kind: SriovNetworkNodePolicy
metadata:
  name: sriov-nnp-fh
  namespace: openshift-sriov-network-operator
  annotations:
    lca.openshift.io/apply-wave: "3"
  spec:
    deviceType: netdevice
    isRdma: true
`

const sriovnetwork1 = `
apiVersion: sriovnetwork.openshift.io/v1
kind: SriovNetwork
metadata:
  name: sriov-nw-mh
  namespace: openshift-sriov-network-operator
  annotations:
    lca.openshift.io/apply-wave: "1"
spec:
  resourceName: mh
`

const sriovnetwork2 = `
apiVersion: sriovnetwork.openshift.io/v1
kind: SriovNetwork
metadata:
  name: sriov-nw-fh
  namespace: openshift-sriov-network-operator-test
  annotations:
    lca.openshift.io/apply-wave: "10"
spec:
  resourceName: fh
`

const srvn_random_chars = `
apiVersion: sriovnetwork.openshift.io/v1
kind: SriovNetwork
metadata: adsafsd
  name: sriov-nw-mh
  namespace: openshift-sriov-network-operator
spec:
  resourceName: mh
`

const srvn_missing_name = `
apiVersion: sriovnetwork.openshift.io/v1
kind: SriovNetwork
spec:
  resourceName: fh
`

const srvn_missing_kind_apiversion = `
apiVersion: sriovnetwork.openshift.io/v1
metadata:
  name: sriov-nw-mh
  namespace: openshift-sriov-network-operator
spec:
  resourceName: fh
---
kind: SriovNetwork
metadata: adsafsd
  name: sriov-nw-mh
  namespace: openshift-sriov-network-operator
spec:
  resourceName: fh
`

const machineConfig = `
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  labels:
    machineconfiguration.openshift.io/role: master
  name: generic
spec:
  config:
    ignition:
      version: 3.2.0
`

const subscription = `
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: local-storage-operator
  namespace: openshift-local-storage
spec:
  channel: "stable"
`

const test_ns = `
apiVersion: v1
kind: Namespace
metadata:
  name: openshift-sriov-network-operator-test
  annotations:
    lca.openshift.io/apply-wave: "1"
`

func TestExportExtraManifests(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()

	// Create a temporary directory for testing
	toDir, err := os.MkdirTemp("", "staterootB")
	if err != nil {
		t.Errorf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(toDir)

	// Create two configmaps
	cms := []*corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "extra-manifest-cm1",
				Namespace: "default",
			},
			Data: map[string]string{
				"sriovnetwork1.yaml": sriovnetwork1,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "extra-manifest-cm2",
				Namespace: "default",
			},
			Data: map[string]string{
				"sriovnetwork2.yaml": sriovnetwork2,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "extra-manifest-cm3",
				Namespace: "default",
			},
			Data: map[string]string{
				"sriovnodepolicies.yaml": sriovnodepolicies,
			},
		},
	}

	for _, cm := range cms {
		err = fakeClient.Create(context.Background(), cm)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	}

	handler := &EMHandler{
		Client:        fakeClient,
		DynamicClient: nil,
		Log:           ctrl.Log.WithName("ExtraManifest"),
	}

	// Export the manifests to the temporary directory
	err = handler.ExportExtraManifestToDir(context.Background(),
		[]ibuv1.ConfigMapRef{
			{Name: "extra-manifest-cm1", Namespace: "default"},
			{Name: "extra-manifest-cm2", Namespace: "default"},
			{Name: "extra-manifest-cm3", Namespace: "default"},
		}, toDir)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check that the manifests were exported to the correct files
	expectedFilePaths := []string{
		filepath.Join(toDir, CmManifestPath, "group2", "1_SriovNetworkNodePolicy_sriov-nnp-fh_openshift-sriov-network-operator.yaml"),
		filepath.Join(toDir, CmManifestPath, "group1", "1_SriovNetworkNodePolicy_sriov-nnp-mh_openshift-sriov-network-operator.yaml"),
		filepath.Join(toDir, CmManifestPath, "group3", "1_SriovNetwork_sriov-nw-fh_openshift-sriov-network-operator-test.yaml"),
		filepath.Join(toDir, CmManifestPath, "group1", "2_SriovNetwork_sriov-nw-mh_openshift-sriov-network-operator.yaml"),
	}

	for _, expectedFile := range expectedFilePaths {
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			t.Fatalf("Expected file %s does not exist", expectedFile)
		}
	}
}

func TestValidateExtraManifestConfigmaps(t *testing.T) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "openshift-sriov-network-operator",
		},
	}

	cms := []client.Object{
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "extra-manifest-cm1-err",
				Namespace: "default",
			},
			Data: map[string]string{
				"sriovnetwork1.yaml":     sriovnetwork1,
				"srvn_random_chars.yaml": srvn_random_chars,
				"srvn_missing_name.yaml": srvn_missing_name,
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "extra-manifest-cm2-err",
				Namespace: "default",
			},
			Data: map[string]string{
				"srvn_missing_kind_apiversion.yaml": srvn_missing_kind_apiversion,
				"generic_mc.yaml":                   machineConfig,
				"subscription.yaml":                 subscription,
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "extra-manifest-cm-warning",
				Namespace: "default",
			},
			Data: map[string]string{
				"sriovnetwork1.yaml":     sriovnetwork1,
				"sriovnetwork2.yaml":     sriovnetwork2,
				"sriovnodepolicies.yaml": sriovnodepolicies,
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "extra-manifest-cm-happy",
				Namespace: "default",
			},
			Data: map[string]string{
				"ns.yaml":            test_ns,
				"sriovnetwork1.yaml": sriovnetwork1,
				"sriovnetwork2.yaml": sriovnetwork2,
			},
		}}

	testcases := []struct {
		name         string
		configmaps   []ibuv1.ConfigMapRef
		expectedErr  error
		expectedWarn string
	}{
		{
			name: "configmap is not found",
			configmaps: []ibuv1.ConfigMapRef{
				{Name: "cm1", Namespace: "default"},
			},
			expectedErr: fmt.Errorf("the extraManifests configMap is not found"),
		},
		{
			name: "validation error with random chars/missing name",
			configmaps: []ibuv1.ConfigMapRef{
				{Name: "extra-manifest-cm1-err", Namespace: "default"},
			},
			expectedErr: fmt.Errorf("failed to decode srvn_random_chars.yaml in the configMap extra-manifest-cm1-err: error converting YAML to JSON: yaml: line 5: mapping values are not allowed in this context; " +
				"SriovNetwork resource name is empty",
			),
			expectedWarn: "",
		},
		{
			name: "validation error with missing Kind/ApiVersion and unsupported resource type",
			configmaps: []ibuv1.ConfigMapRef{
				{Name: "extra-manifest-cm2-err", Namespace: "default"},
			},
			expectedErr: fmt.Errorf("failed to decode srvn_missing_kind_apiversion.yaml in the configMap extra-manifest-cm2-err: error unmarshaling JSON: while decoding JSON: Object 'Kind' is missing in '{\"apiVersion\":\"sriovnetwork.openshift.io/v1\",\"metadata\":{\"name\":\"sriov-nw-mh\",\"namespace\":\"openshift-sriov-network-operator\"},\"spec\":{\"resourceName\":\"fh\"}}'; " +
				"failed to decode srvn_missing_kind_apiversion.yaml in the configMap extra-manifest-cm2-err: error converting YAML to JSON: yaml: line 3: mapping values are not allowed in this context; " +
				"MachineConfig[generic] in extramanifests is not allowed; " +
				"Subscription[local-storage-operator] in extramanifests is not allowed"),
			expectedWarn: "",
		},
		{
			name: "validation warning with unknown namespace and CRD",
			configmaps: []ibuv1.ConfigMapRef{
				{Name: "extra-manifest-cm-warning", Namespace: "default"},
			},
			expectedErr: nil,
			expectedWarn: "SriovNetworkNodePolicy[sriov-nnp-mh] - CRD not deployed on cluster; " +
				"SriovNetworkNodePolicy[sriov-nnp-fh] - CRD not deployed on cluster; " +
				"SriovNetwork[sriov-nw-fh] - namespace openshift-sriov-network-operator-test not found on cluster or not added in configmap in right order",
		},
		{
			name: "validation pass with ns added in configmap",
			configmaps: []ibuv1.ConfigMapRef{
				{Name: "extra-manifest-cm-happy", Namespace: "default"},
			},
			expectedErr:  nil,
			expectedWarn: "",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			objs := []client.Object{ns}
			objs = append(objs, cms...)

			mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{sriovv1.GroupVersion})
			mapper.Add(schema.GroupVersionKind{Group: "sriovnetwork.openshift.io", Version: "v1", Kind: "SriovNetwork"}, meta.RESTScopeNamespace)
			fakeClient := fake.NewClientBuilder().WithScheme(testscheme).WithRESTMapper(mapper).WithObjects(objs...).Build()

			dynamicFakeClient := dynamicfake.NewSimpleDynamicClient(scheme.Scheme)

			handler := &EMHandler{
				Client:        fakeClient,
				DynamicClient: dynamicFakeClient,
				Log:           ctrl.Log.WithName("ExtraManifest"),
			}

			warn, err := handler.ValidateExtraManifestConfigmaps(context.Background(), tc.configmaps)
			if err != nil && tc.expectedErr == nil {
				t.Errorf("Unexpected error: %v", err)
			} else if err == nil && tc.expectedErr != nil {
				t.Errorf("Expected error not found: %s", tc.expectedErr)
			} else if err != nil && tc.expectedErr != nil {
				assert.ErrorContains(t, err, tc.expectedErr.Error())
			}
			assert.Equal(t, tc.expectedWarn, warn)
		})
	}
}

const policyWithLabelAndAnnotation = `---
kind: Policy
apiVersion: policy.open-cluster-management.io/v1
metadata:
  name: ztp-common.p1
  namespace: spoke
  labels:
    lca.openshift.io/target-ocp-version: "4.16.1"
  annotations:
    ran.openshift.io/ztp-deploy-wave: "1"
spec:
  disabled: false
  policy-templates:
    - objectDefinition:
        apiVersion: policy.open-cluster-management.io/v1
        kind: ConfigurationPolicy
        metadata:
          name: common-cnfdf22-new-config-policy-config
        spec:
          evaluationInterval:
            compliant: 10m
            noncompliant: 10s
          namespaceselector:
            exclude:
              - kube-*
            include:
              - "*"
          object-templates:
            - complianceType: musthave
              objectDefinition:
                apiVersion: operators.coreos.com/v1alpha1
                kind: CatalogSource
                metadata:
                  name: redhat-operators-new
                  namespace: openshift-marketplace
                  annotations:
                    target.workload.openshift.io/management: '{"effect": "PreferredDuringScheduling"}'
                spec:
                  displayName: Red Hat Operators Catalog
                  image: registry.redhat.io/redhat/redhat-operator-index:v4.16
                  publisher: Red Hat
                  sourceType: grpc
                  updateStrategy:
                    registryPoll:
                      interval: 1h
                status:
                  connectionState:
                    lastObservedState: READY
          remediationAction: inform
          severity: low
  remediationAction: inform
`

const policyWithAnnotationAndObjectWithLabel = `---
kind: Policy
apiVersion: policy.open-cluster-management.io/v1
metadata:
  name: ztp-common.p3
  namespace: spoke
  annotations:
    ran.openshift.io/ztp-deploy-wave: "1"
spec:
  disabled: false
  policy-templates:
    - objectDefinition:
        apiVersion: policy.open-cluster-management.io/v1
        kind: ConfigurationPolicy
        metadata:
          name: common-cnfdf22-new-config-policy-config
        spec:
          evaluationInterval:
            compliant: 10m
            noncompliant: 10s
          namespaceselector:
            exclude:
              - kube-*
            include:
              - "*"
          object-templates:
            - complianceType: musthave
              objectDefinition:
                apiVersion: operators.coreos.com/v1alpha1
                kind: CatalogSource
                metadata:
                  name: redhat-operators-new
                  namespace: openshift-marketplace
                  annotations:
                    target.workload.openshift.io/management: '{"effect": "PreferredDuringScheduling"}'
                  labels:
                    lca.openshift.io/target-ocp-version: "4.15.2"
                spec:
                  displayName: Red Hat Operators Catalog
                  image: registry.redhat.io/redhat/redhat-operator-index:v4.15
                  publisher: Red Hat
                  sourceType: grpc
                  updateStrategy:
                    registryPoll:
                      interval: 1h
                status:
                  connectionState:
                    lastObservedState: READY
          remediationAction: inform
          severity: low
  remediationAction: inform
`

const policyWithAnnotationAndObjectWithNonMatchedLabel = `---
kind: Policy
apiVersion: policy.open-cluster-management.io/v1
metadata:
  name: ztp-common.p4
  namespace: spoke
  annotations:
    ran.openshift.io/ztp-deploy-wave: "2"
spec:
  disabled: false
  policy-templates:
    - objectDefinition:
        apiVersion: policy.open-cluster-management.io/v1
        kind: ConfigurationPolicy
        metadata:
          name: common-cnfdf22-new-config-policy-config
        spec:
          evaluationInterval:
            compliant: 10m
            noncompliant: 10s
          namespaceselector:
            exclude:
              - kube-*
            include:
              - "*"
          object-templates:
            - complianceType: musthave
              objectDefinition:
                apiVersion: operators.coreos.com/v1alpha1
                kind: CatalogSource
                metadata:
                  name: redhat-operators-non-match
                  namespace: openshift-marketplace
                  annotations:
                    target.workload.openshift.io/management: '{"effect": "PreferredDuringScheduling"}'
                  labels:
                    lca.openshift.io/target-ocp-version: "4.15.6"
                spec:
                  displayName: Red Hat Operators Catalog
                  image: registry.redhat.io/redhat/redhat-operator-index:v4.15
                  publisher: Red Hat
                  sourceType: grpc
                  updateStrategy:
                    registryPoll:
                      interval: 1h
                status:
                  connectionState:
                    lastObservedState: READY
          remediationAction: inform
          severity: low
  remediationAction: inform
`

const policyWithoutLabel = `---
kind: Policy
apiVersion: policy.open-cluster-management.io/v1
metadata:
  name: ztp-group.p2
  namespace: spoke
spec:
  disabled: false
  policy-templates:
    - objectDefinition:
        apiVersion: policy.open-cluster-management.io/v1
        kind: ConfigurationPolicy
        metadata:
          name: common-cnfdf22-new-config-policy-config
        spec:
          evaluationInterval:
            compliant: 10m
            noncompliant: 10s
          namespaceselector:
            exclude:
              - kube-*
            include:
              - "*"
          object-templates:
            - complianceType: musthave
              objectDefinition:
                apiVersion: operators.coreos.com/v1alpha1
                kind: CatalogSource
                metadata:
                  name: redhat-operators
                  namespace: openshift-marketplace
                  annotations:
                    target.workload.openshift.io/management: '{"effect": "PreferredDuringScheduling"}'
                spec:
                  displayName: Red Hat Operators Catalog
                  image: registry.redhat.io/redhat/redhat-operator-index:v4.14
                  publisher: Red Hat
                  sourceType: grpc
                  updateStrategy:
                    registryPoll:
                      interval: 1h
                status:
                  connectionState:
                    lastObservedState: READY
          remediationAction: inform
          severity: low
  remediationAction: inform
`

func TestExportPolicyManifests(t *testing.T) {

	testcases := []struct {
		name                string
		policies            []*unstructured.Unstructured
		policyLabels        map[string]string
		objectLabels        map[string]string
		validationAnns      map[string]string
		expectedFilePaths   []string
		unexpectedFilePaths []string
		expectedObjects     []unstructured.Unstructured
		expectedError       string
	}{
		{
			name: "happy path with validation and extraction by matching labels on objects",
			policies: []*unstructured.Unstructured{
				mustConvertYamlStrToUnstructured(policyWithAnnotationAndObjectWithLabel),
				mustConvertYamlStrToUnstructured(policyWithoutLabel),
				mustConvertYamlStrToUnstructured(policyWithAnnotationAndObjectWithNonMatchedLabel),
			},
			objectLabels:   map[string]string{TargetOcpVersionLabel: "4.15.2-rc1,4.15.2,4.15"},
			validationAnns: map[string]string{TargetOcpVersionManifestCountAnnotation: "1"},
			expectedFilePaths: []string{
				filepath.Join(PolicyManifestPath, "group1", "1_CatalogSource_redhat-operators-new_openshift-marketplace.yaml"),
			},
			unexpectedFilePaths: []string{
				filepath.Join(PolicyManifestPath, "group2", "1_CatalogSource_redhat-operators-non-match_openshift-marketplace.yaml"),
				filepath.Join(PolicyManifestPath, "group1", "2_CatalogSource_redhat-operators_openshift-marketplace.yaml"),
			},
			expectedObjects: []unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"apiVersion": "operators.coreos.com/v1alpha1",
						"kind":       "CatalogSource",
						"metadata": map[string]interface{}{
							"annotations": map[string]interface{}{
								"target.workload.openshift.io/management": "{\"effect\": \"PreferredDuringScheduling\"}",
								common.ApplyTypeAnnotation:                common.ApplyTypeMerge,
							},
							"labels": map[string]interface{}{
								"lca.openshift.io/target-ocp-version": "4.15.2",
							},
							"name":      "redhat-operators-new",
							"namespace": "openshift-marketplace",
						},
						"spec": map[string]interface{}{
							"displayName": "Red Hat Operators Catalog",
							"image":       "registry.redhat.io/redhat/redhat-operator-index:v4.15",
							"publisher":   "Red Hat",
							"sourceType":  "grpc",
							"updateStrategy": map[string]interface{}{
								"registryPoll": map[string]interface{}{
									"interval": "1h",
								},
							},
						},
						// status should be empty
						"status": map[string]interface{}{},
					},
				},
			},
		},
		{
			name:           "happy path with validation and extraction by matching labels on policies",
			policies:       []*unstructured.Unstructured{mustConvertYamlStrToUnstructured(policyWithLabelAndAnnotation), mustConvertYamlStrToUnstructured(policyWithoutLabel)},
			policyLabels:   map[string]string{TargetOcpVersionLabel: "4.16.1"},
			validationAnns: map[string]string{TargetOcpVersionManifestCountAnnotation: "1"},
			expectedFilePaths: []string{
				filepath.Join(PolicyManifestPath, "group1", "1_CatalogSource_redhat-operators-new_openshift-marketplace.yaml"),
			},
			unexpectedFilePaths: []string{
				filepath.Join(PolicyManifestPath, "group1", "1_CatalogSource_redhat-operators_openshift-marketplace.yaml"),
			},
			expectedObjects: []unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"apiVersion": "operators.coreos.com/v1alpha1",
						"kind":       "CatalogSource",
						"metadata": map[string]interface{}{
							"annotations": map[string]interface{}{
								"target.workload.openshift.io/management": "{\"effect\": \"PreferredDuringScheduling\"}",
							},
							"name":      "redhat-operators-new",
							"namespace": "openshift-marketplace",
						},
						"spec": map[string]interface{}{
							"displayName": "Red Hat Operators Catalog",
							"image":       "registry.redhat.io/redhat/redhat-operator-index:v4.16",
							"publisher":   "Red Hat",
							"sourceType":  "grpc",
							"updateStrategy": map[string]interface{}{
								"registryPoll": map[string]interface{}{
									"interval": "1h",
								},
							},
						},
						// status should be empty
						"status": map[string]interface{}{},
					},
				},
			},
		},
		{
			name:              "manifests count does not match with expected",
			policies:          []*unstructured.Unstructured{mustConvertYamlStrToUnstructured(policyWithAnnotationAndObjectWithLabel)},
			objectLabels:      map[string]string{TargetOcpVersionLabel: "4.15.2-rc1,4.15.2,4.15"},
			validationAnns:    map[string]string{TargetOcpVersionManifestCountAnnotation: "3"},
			expectedFilePaths: []string{},
			expectedError:     "does not match the expected manifests count",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().Build()

			// Create a temporary directory for testing
			toDir, err := os.MkdirTemp("", "staterootB")
			if err != nil {
				t.Errorf("Failed to create temporary directory: %v", err)
			}
			defer os.RemoveAll(toDir)

			crd := &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "policies.policy.open-cluster-management.io",
				},
			}

			err = fakeClient.Create(context.Background(), crd)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			for _, p := range tc.policies {
				err = fakeClient.Create(context.Background(), p)
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			handler := &EMHandler{
				Client: fakeClient,
				Log:    ctrl.Log.WithName("ExtraManifest"),
			}

			// Export the manifests to the temporary directory
			err = handler.ExtractAndExportManifestFromPoliciesToDir(context.Background(),
				tc.policyLabels, tc.objectLabels, tc.validationAnns, toDir)
			if err != nil {
				if tc.expectedError != "" {
					assert.Contains(t, err.Error(), tc.expectedError)
				} else {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			// Check that the manifests were exported to the correct files

			for i, expectedFile := range tc.expectedFilePaths {
				if _, err := os.Stat(filepath.Join(toDir, expectedFile)); os.IsNotExist(err) {
					t.Fatalf("Expected file %s does not exist", expectedFile)
				}
				object := &unstructured.Unstructured{}
				err := utils.ReadYamlOrJSONFile(filepath.Join(toDir, expectedFile), object)
				if err != nil {
					t.Fatalf("Failed to read expected file %s, err: %v", expectedFile, err)
				}
				if !equality.Semantic.DeepEqual(object, &tc.expectedObjects[i]) {
					t.Fatalf("exported CR: \n%v does not match expected: \n%v", object, tc.expectedObjects[i])
				}

			}
			for _, unexpectedFile := range tc.unexpectedFilePaths {
				if _, err := os.Stat(filepath.Join(toDir, unexpectedFile)); !os.IsNotExist(err) {
					t.Fatalf("Unexpected file %s should not exist", unexpectedFile)
				}
			}
		})
	}
}

// convertYamlStrToUnstructured helper func to convert a CR in Yaml string to Unstructured
func mustConvertYamlStrToUnstructured(cr string) *unstructured.Unstructured {
	jCr, err := yaml.ToJSON([]byte(cr))
	if err != nil {
		panic(err.Error())
	}

	object, err := runtime.Decode(unstructured.UnstructuredJSONScheme, jCr)
	if err != nil {
		panic(err.Error())
	}

	uCr, ok := object.(*unstructured.Unstructured)
	if !ok {
		panic("unstructured.Unstructured expected")
	}
	return uCr
}

func TestGetMatchingVersion(t *testing.T) {
	tests := []struct {
		targetOCPversion string
		expected         []string
	}{
		{
			targetOCPversion: "4.15.2-ec.3",
			expected:         []string{"4.15.2-ec.3", "4.15.2", "4.15"},
		},
		{
			targetOCPversion: "4.15.2",
			expected:         []string{"4.15.2", "4.15"},
		},
		{
			targetOCPversion: "4.16.0-0.ci-2024-04-11-051453",
			expected:         []string{"4.16.0-0.ci-2024-04-11-051453", "4.16.0", "4.16"},
		},
	}

	t.Run("TargetOCPversion test", func(t *testing.T) {
		for _, tc := range tests {
			result, err := GetMatchingTargetOcpVersionLabelVersions(tc.targetOCPversion)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			assert.ElementsMatch(t, tc.expected, result)
		}
	})
}

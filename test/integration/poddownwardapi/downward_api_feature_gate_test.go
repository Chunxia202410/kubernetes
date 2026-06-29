/*
Copyright 2025 The Kubernetes Authors.

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

package poddownwardapi

import (
	"context"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	clientset "k8s.io/client-go/kubernetes"
	featuregatetesting "k8s.io/component-base/featuregate/testing"
	kubeapiservertesting "k8s.io/kubernetes/cmd/kube-apiserver/app/testing"
	"k8s.io/kubernetes/pkg/features"
	"k8s.io/kubernetes/test/integration/framework"
)

// makePodWithAssignedCpusetDownwardAPI creates a pod spec that includes a DownwardAPI volume
// with an assigned.cpuset resource field reference. This is the field that is gated by
// the DownwardAPIAssignedResources feature gate.
func makePodWithAssignedCpusetDownwardAPI(name string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "test-container",
					Image: "fakeimage",
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      "podinfo",
							MountPath: "/podinfo",
						},
					},
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("100m"),
							v1.ResourceMemory: resource.MustParse("100Mi"),
						},
					},
				},
			},
			Volumes: []v1.Volume{
				{
					Name: "podinfo",
					VolumeSource: v1.VolumeSource{
						DownwardAPI: &v1.DownwardAPIVolumeSource{
							Items: []v1.DownwardAPIVolumeFile{
								{
									Path: "assigned_cpuset_test-container",
									ResourceFieldRef: &v1.ResourceFieldSelector{
										ContainerName: "test-container",
										Resource:      "assigned.cpuset",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// hasAssignedCpusetDownwardAPIItem checks if the pod spec contains the assigned.cpuset
// DownwardAPI volume file item.
func hasAssignedCpusetDownwardAPIItem(pod *v1.Pod) bool {
	for _, vol := range pod.Spec.Volumes {
		if vol.DownwardAPI != nil {
			for _, item := range vol.DownwardAPI.Items {
				if item.ResourceFieldRef != nil && item.ResourceFieldRef.Resource == "assigned.cpuset" {
					return true
				}
			}
		}
	}
	return false
}

// TestDownwardAPIAssignedResourcesGateOff verifies that when the DownwardAPIAssignedResources
// feature gate is disabled, the assigned.cpuset DownwardAPI volume file items are dropped
// from the pod spec by the API server.
func TestDownwardAPIAssignedResourcesGateOff(t *testing.T) {
	// Enable the feature gate BEFORE starting the test server, as the API server
	// checks feature gates on start up and not on each invocation at runtime.
	featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.DownwardAPIAssignedResources, false)

	server := kubeapiservertesting.StartTestServerOrDie(t, nil, []string{"--disable-admission-plugins=ServiceAccount"}, framework.SharedEtcd())
	defer server.TearDownFn()

	client := clientset.NewForConfigOrDie(server.ClientConfig)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a namespace
	ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-downwardapi-off"}}
	_, err := client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create namespace: %v", err)
	}

	// Create a pod with assigned.cpuset DownwardAPI item
	pod := makePodWithAssignedCpusetDownwardAPI("test-pod-gate-off")
	createdPod, err := client.CoreV1().Pods("test-downwardapi-off").Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create pod: %v", err)
	}

	// Verify that assigned.cpuset DownwardAPI item was DROPPED
	if hasAssignedCpusetDownwardAPIItem(createdPod) {
		t.Errorf("Expected assigned.cpuset DownwardAPI item to be dropped when feature gate is OFF, but it was preserved")
	}

	// Verify the DownwardAPI volume still exists but with empty items
	foundDownwardAPIVolume := false
	for _, vol := range createdPod.Spec.Volumes {
		if vol.DownwardAPI != nil {
			foundDownwardAPIVolume = true
			if len(vol.DownwardAPI.Items) != 0 {
				t.Errorf("Expected DownwardAPI volume to have 0 items when feature gate is OFF, got %d items", len(vol.DownwardAPI.Items))
			}
		}
	}
	if !foundDownwardAPIVolume {
		t.Error("Expected DownwardAPI volume to still exist (with empty items), but it was not found")
	}

	t.Logf("PASS: assigned.cpuset DownwardAPI item was correctly dropped when feature gate is OFF")
}

// TestDownwardAPIAssignedResourcesGateOn verifies that when the DownwardAPIAssignedResources
// feature gate is enabled, the assigned.cpuset DownwardAPI volume file items are preserved
// in the pod spec by the API server.
func TestDownwardAPIAssignedResourcesGateOn(t *testing.T) {
	// Enable the feature gate BEFORE starting the test server
	featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.DownwardAPIAssignedResources, true)

	server := kubeapiservertesting.StartTestServerOrDie(t, nil, []string{"--disable-admission-plugins=ServiceAccount"}, framework.SharedEtcd())
	defer server.TearDownFn()

	client := clientset.NewForConfigOrDie(server.ClientConfig)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a namespace
	ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-downwardapi-on"}}
	_, err := client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create namespace: %v", err)
	}

	// Create a pod with assigned.cpuset DownwardAPI item
	pod := makePodWithAssignedCpusetDownwardAPI("test-pod-gate-on")
	createdPod, err := client.CoreV1().Pods("test-downwardapi-on").Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create pod: %v", err)
	}

	// Verify that assigned.cpuset DownwardAPI item was PRESERVED
	if !hasAssignedCpusetDownwardAPIItem(createdPod) {
		t.Errorf("Expected assigned.cpuset DownwardAPI item to be preserved when feature gate is ON, but it was dropped")
	}

	// Verify the specific details of the preserved item
	foundItem := false
	for _, vol := range createdPod.Spec.Volumes {
		if vol.DownwardAPI != nil {
			for _, item := range vol.DownwardAPI.Items {
				if item.ResourceFieldRef != nil && item.ResourceFieldRef.Resource == "assigned.cpuset" {
					foundItem = true
					if item.Path != "assigned_cpuset_test-container" {
						t.Errorf("Expected DownwardAPI item path 'assigned_cpuset_test-container', got %q", item.Path)
					}
					if item.ResourceFieldRef.ContainerName != "test-container" {
						t.Errorf("Expected DownwardAPI item container name 'test-container', got %q", item.ResourceFieldRef.ContainerName)
					}
				}
			}
		}
	}
	if !foundItem {
		t.Error("Expected to find assigned.cpuset DownwardAPI item with correct details, but it was not found")
	}

	t.Logf("PASS: assigned.cpuset DownwardAPI item was correctly preserved when feature gate is ON")
}

// TestDownwardAPIAssignedResourcesToggleOnOff verifies the rollback scenario:
// 1. Create a pod with the feature gate ON (items preserved)
// 2. Disable the feature gate
// 3. Read the pod back, items should still be preserved (seamless rollback)
// 4. Deploy a new pod, items should drop now (rollout)
func TestDownwardAPIAssignedResourcesToggleOnOff(t *testing.T) {
	// Step 1: Start with feature gate ON
	featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.DownwardAPIAssignedResources, true)

	server := kubeapiservertesting.StartTestServerOrDie(t, nil, []string{"--disable-admission-plugins=ServiceAccount"}, framework.SharedEtcd())
	defer server.TearDownFn()

	client := clientset.NewForConfigOrDie(server.ClientConfig)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a namespace
	ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-downwardapi-toggle"}}
	_, err := client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create namespace: %v", err)
	}

	// Create a pod with assigned.cpuset DownwardAPI item (gate ON)
	pod := makePodWithAssignedCpusetDownwardAPI("test-pod-toggle")
	createdPod, err := client.CoreV1().Pods("test-downwardapi-toggle").Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create pod: %v", err)
	}

	// Verify items are preserved when gate is ON
	if !hasAssignedCpusetDownwardAPIItem(createdPod) {
		t.Fatalf("Expected assigned.cpuset DownwardAPI item to be preserved when feature gate is ON, but it was dropped")
	}
	t.Logf("Step 1 PASS: assigned.cpuset DownwardAPI item preserved when gate is ON")

	// Step 2: Toggle the feature gate OFF
	// Note: In integration tests, the API server runs in-process and shares the same
	// DefaultFeatureGate. Toggling the gate affects the API server's behavior.
	// TODO: Need to verify if we can avoid toggling the feature-gate any other way.
	utilfeature.DefaultMutableFeatureGate.Set("DownwardAPIAssignedResources=false")
	t.Logf("Step 2: Feature gate toggled OFF")

	// Step 3: Read the pod back - items should still be preserved (rollback safe)
	readPod, err := client.CoreV1().Pods("test-downwardapi-toggle").Get(ctx, createdPod.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get pod: %v", err)
	}

	// The assigned.cpuset items should still be present in the stored pod spec
	// because dropDisabledAssignedCpuset only drops items on CREATE/UPDATE,
	// not on READ. The data is already in etcd.
	if !hasAssignedCpusetDownwardAPIItem(readPod) {
		t.Errorf("Expected assigned.cpuset DownwardAPI item to still be present after toggling gate OFF (rollback safe), but it was dropped")
	} else {
		t.Logf("Step 3 PASS: assigned.cpuset DownwardAPI item still present after gate OFF (rollback safe)")
	}

	// Step 4: Create a NEW pod with gate OFF - items should be dropped
	newPod := makePodWithAssignedCpusetDownwardAPI("test-pod-new-gate-off")
	createdNewPod, err := client.CoreV1().Pods("test-downwardapi-toggle").Create(ctx, newPod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create new pod with gate OFF: %v", err)
	}

	if hasAssignedCpusetDownwardAPIItem(createdNewPod) {
		t.Errorf("Expected assigned.cpuset DownwardAPI item to be dropped for NEW pod when gate is OFF, but it was preserved")
	} else {
		t.Logf("Step 4 PASS: assigned.cpuset DownwardAPI item correctly dropped for NEW pod when gate is OFF")
	}
}

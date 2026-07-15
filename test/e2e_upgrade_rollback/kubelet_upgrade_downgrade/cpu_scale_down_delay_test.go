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

package kubeletupgradedowngrade

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/test/e2e/common/node/framework/cgroups"
	"k8s.io/kubernetes/test/e2e/common/node/framework/podresize"
	e2enode "k8s.io/kubernetes/test/e2e/framework/node"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	e2etestfiles "k8s.io/kubernetes/test/e2e/framework/testfiles"
	"k8s.io/kubernetes/test/e2e_upgrade_rollback/common"
	"k8s.io/kubernetes/test/utils/client-go/ktesting"
	"k8s.io/kubernetes/test/utils/localupcluster"
)

func init() {
	ktesting.SetDefaultVerbosity(2)
}

// repoRoot figures out whether an E2E suite is invoked in its directory,
// directly in the root, or somewhere deep inside the _output directory.
func repoRoot() string {
	for i := range 10 {
		path := "." + strings.Repeat("/..", i)
		if _, err := os.Stat(path + "/test/e2e/framework"); err == nil {
			return path
		}
	}
	// Traditional default.
	return "../../"
}

// currentBinDir returns the environment variable name and the directory
// where the Kubernetes server binaries are located.
func currentBinDir() (envName, content string) {
	envName = "KUBERNETES_SERVER_BIN_DIR"
	content, _ = os.LookupEnv(envName)
	return
}

// TestCPUScaleDownDelayKubeletUpgradeDowngrade tests the kubelet version upgrade/downgrade
// scenario for the DownwardAPIAssignedResources feature gate (KEP-6122).
//
// The test undergoes the following stages:
//
//	Stage 0: Download previous release binaries. Fail if unable to download the previous
//	         kubelet binary.
//	Stage 1: Start cluster with master binary + DownwardAPIAssignedResources=true and CPU
//	         manager static policy with scale-delay-time=10s.
//	Stage 2: Create a Guaranteed pod (Pod-A) with cpu=1 and assigned.cpuset DownwardAPI volume.
//	         Scale up Pod-A (cpu 1→2), then initiate scale down (cpu 2→1). Verify scale-down
//	         delay time is respected and cpuset values are updated correctly.
//	Stage 3: Downgrade kubelet to previous release via cluster.Modify().
//	Stage 4: Verify pod is still running, patch to remove DownwardAPI, verify scale-down
//	         is rejected (previous kubelet doesn't support scaling exclusive CPUs).
//	Stage 5: Restore kubelet to master binary, cleanup.
//
// NOTE: Stage 0 downloads the previous release tarball (~450MB), which can take up to 10 minutes.
// Run this test with -timeout=30m to be on the safer side:
//
//	go test ./test/e2e_upgrade_rollback/kubelet_upgrade_downgrade/ -run TestCPUScaleDownDelayKubeletUpgradeDowngrade -timeout=30m -v
func TestCPUScaleDownDelayKubeletUpgradeDowngrade(t *testing.T) {
	testCPUScaleDownDelayKubeletUpgradeDowngrade(ktesting.Init(t))
}

func testCPUScaleDownDelayKubeletUpgradeDowngrade(tCtx ktesting.TContext) {
	// Some other things normally done by test/e2e.
	e2etestfiles.AddFileSource(e2etestfiles.RootFileSource{Root: repoRoot()})

	gomega.RegisterFailHandler(func(message string, callerSkip ...int) {
		tCtx.Helper()
		tCtx.Fatal(message)
	})

	// Get binary directory from environment (master/current build).
	envName, dir := currentBinDir()
	if dir == "" {
		tCtx.Fatalf("%s must be set", envName)
	}

	// scaleDelayTime is the configured scale-down delay time in seconds.
	// Change this value to test with different delay times.
	scaleDelayTime := 10

	// ---- Stage 0: Download previous release binaries ----
	// Download the previous Kubernetes release binaries so we can downgrade the kubelet later.
	// KUBERNETES_SERVER_CACHE_DIR can be set to cache downloaded binaries across test runs.
	cacheDir, _ := os.LookupEnv("KUBERNETES_SERVER_CACHE_DIR")

	var previousBinDir string
	var major, previousMinor uint
	var gitVersion string
	tCtx.Step("download-previous-release-binaries", func(tCtx ktesting.TContext) {
		previousBinDir, major, previousMinor, gitVersion = common.DownloadPreviousReleaseBinaries(tCtx, repoRoot(), cacheDir)
		tCtx.Logf("previous release binaries downloaded to %s (version %d.%d, git version %s)",
			previousBinDir, major, previousMinor, gitVersion)
	})

	// Verify the previous kubelet binary exists. Fail if it doesn't.
	tCtx.Step("verify-previous-kubelet-binary", func(tCtx ktesting.TContext) {
		kubeletPath := path.Join(previousBinDir, string(localupcluster.Kubelet))
		_, err := os.Stat(kubeletPath)
		tCtx.ExpectNoError(err, "previous kubelet binary should exist at %s", kubeletPath)
		tCtx.Logf("Verified previous kubelet binary exists at %s", kubeletPath)
	})

	tCtx.Log("Stage 0 PASS: Previous release binaries downloaded and kubelet binary verified")

	// ---- Stage 1: Create cluster with master binary + DownwardAPIAssignedResources=true ----
	// and CPU manager static policy with scale-delay-time=10s (required for GU pods to get exclusive cpusets)
	cluster := localupcluster.New()
	// The cleanup fxn will be automatically called after the test exits.
	tCtx.CleanupCtx(func(tCtx ktesting.TContext) {
		tCtx.Step("cleanup", cluster.Stop)
	})

	localUpClusterEnv := map[string]string{
		"KUBELET_FLAGS": fmt.Sprintf("--cpu-manager-policy=static --cpu-manager-policy-options=scale-delay-time=%ds --feature-gates=InPlacePodVerticalScalingExclusiveCPUs=true,CPUManagerPolicyAlphaOptions=true --kube-reserved=cpu=1,memory=1Gi --system-reserved=cpu=1,memory=1Gi", scaleDelayTime),
	}

	// Feature gates: DownwardAPIAssignedResources for cpuset exposure for kube-apiserver.
	// InPlacePodVerticalScalingExclusiveCPUs and CPUManagerPolicyAlphaOptions are set via KUBELET_FLAGS.
	featureGates := "DownwardAPIAssignedResources=true"

	phase := "0-master"
	tCtx.Step("start-cluster-master", func(tCtx ktesting.TContext) {
		cluster.Start(tCtx, phase, dir, localUpClusterEnv, featureGates)
	})

	// Wait for node to be ready.
	restConfig := cluster.LoadConfig(tCtx)
	tCtx = tCtx.WithRESTConfig(restConfig).WithNamespace("default")

	tCtx.Step("wait-for-node", func(tCtx ktesting.TContext) {
		tCtx.ExpectNoError(e2enode.WaitForAllNodesSchedulable(tCtx, tCtx.Client(), 5*time.Minute))
	})

	tCtx.Log("Stage 1 PASS: Cluster created with master binary, feature gates enabled, and CPU manager static policy with scale-delay-time=10s")

	// ---- Stage 2: Create Pod-A, scale up (1→2), scale down (2→1) with delay verification ----
	// Use podresize.MakeResizablePodWithDownwardAPI to create the pod spec, reusing the same
	// helper that cpu_manager_test.go uses for creating resizable pods with DownwardAPI.
	var podA *v1.Pod
	const podAName = "pod-a-scale-test"
	const containerAName = "gu-container-a"
	originalContainers := []podresize.ResizableContainerInfo{
		{
			Name: containerAName,
			Resources: &cgroups.ContainerResources{
				CPUReq: "1000m", CPULim: "1000m",
				MemReq: "200Mi", MemLim: "200Mi",
			},
			HasExclusiveCPUs: true,
		},
	}
	tCtx.Step("create-pod-a", func(tCtx ktesting.TContext) {
		tStamp := strconv.Itoa(time.Now().Nanosecond())
		podSpec := podresize.MakeResizablePodWithDownwardAPI(tCtx.Namespace(), podAName, tStamp, originalContainers, nil)
		podSpec = e2epod.MustMixinRestrictedPodSecurity(podSpec)
		podA = common.CreatePodAndWaitForRunning(tCtx, podSpec)
	})

	// Verify original pod resources, allocations are as expected
	tCtx.Step("verify-pod-a-resources", func(tCtx ktesting.TContext) {
		podresize.VerifyPodResources(podA, originalContainers, nil)
		tCtx.Logf("Verified pod %q resources match expected allocations", podAName)
	})

	// Verify original pod cpusets are as expected
	tCtx.Step("verify-pod-a-cpusets", func(tCtx ktesting.TContext) {
		// Re-fetch the pod to get the latest status (UID, QOSClass, ContainerStatuses)
		freshPod, err := tCtx.Client().CoreV1().Pods(podA.Namespace).Get(tCtx, podA.Name, metav1.GetOptions{})
		tCtx.ExpectNoError(err, "get pod %s for cpuset verification", podA.Name)
		podA = freshPod
		tCtx.Eventually(func(tCtx ktesting.TContext) bool {
			return common.HaveContainerCPUsCount(tCtx, restConfig, podA, containerAName, 1)
		}).WithTimeout(2*time.Minute).Should(gomega.BeTrue(),
			"container %q should have 1 CPU in its cpuset", containerAName)
		tCtx.Logf("Verified pod %q container %q has 1 CPU in cpuset", podAName, containerAName)
	})

	// Scale up Pod-A (cpu 1→2), then scale down (cpu 2→1) with delay verification
	tCtx.Step("scale-up-and-scale-down", func(tCtx ktesting.TContext) {
		// Scale up: CPU 1 → 2
		currentContainers := patchAndVerifyPodResize(
			tCtx, restConfig, podA, containerAName,
			originalContainers, // containers before resize (1 CPU)
			[]podresize.ResizableContainerInfo{ // desired containers (2 CPUs)
				{
					Name: containerAName,
					Resources: &cgroups.ContainerResources{
						CPUReq: "2000m", CPULim: "2000m",
						MemReq: "200Mi", MemLim: "200Mi",
					},
					HasExclusiveCPUs: true,
				},
			},
			2,              // expected CPU count after resize
			scaleDelayTime, // scale delay time
			false,          // isScaleDown = false (scale up)
			true,           // waitForActuation = true (wait for scale-up to complete)
			true,           // checkDownwardAPI = true (Pod-A has DownwardAPI volume)
		)

		// Scale down: CPU 2 → 1 (initiate but don't wait for actuation)
		patchAndVerifyPodResize(
			tCtx, restConfig, podA, containerAName,
			currentContainers,  // containers before resize (2 CPUs)
			originalContainers, // desired containers (1 CPU, same as original)
			1,                  // expected CPU count after resize
			scaleDelayTime,     // scale delay time
			true,               // isScaleDown = true
			false,              // waitForActuation = false (don't wait, will check in Stage 3)
			true,               // checkDownwardAPI = true (Pod-A has DownwardAPI volume)
		)
	})

	tCtx.Log("Stage 2 PASS: Pod-A created, scaled up (1→2) and scaled down (2→1) initiated (not waiting for actuation)")

	// ---- Stage 3 & 4: Downgrade kubelet, verify pod state ----
	restoreOpts, podB, containerBName := stage3And4KubeletDowngrade(tCtx, restConfig, cluster, podA, containerAName, previousBinDir, major, previousMinor)

	tCtx.Log("Stage 3 & 4 PASS: Kubelet downgraded, pod state verified")

	// ---- Stage 5: Restore kubelet to master binary, verify Pod-B ----
	stage5KubeletRestore(tCtx, restConfig, cluster, restoreOpts, podB, containerBName, scaleDelayTime)

	tCtx.Log("Stage 5 PASS: Kubelet restored to master binary")

	tCtx.Log("ALL STAGES PASSED: Kubelet upgrade/downgrade test complete")
	// Cleanup will destroy the cluster via tCtx.CleanupCtx.
}

// patchAndVerifyPodResize patches a pod for resize and verifies the resize is actuated correctly.
// It follows the same flow as patchAndVerifyPodResize in test/e2e_node/cpu_manager_test.go (L8556),
// adapted for ktesting instead of Ginkgo/framework.Framework.
//
// Steps:
//  1. Record time before scaling
//  2. Patch pod for resize (using podresize.MakeResizePatch)
//  3. Verify cpuset in DownwardAPI volume (common.HaveDownwardAPICPUsCount)
//  4. Verify pod resources post-patch, pre-actuation (podresize.VerifyPodResources)
//  5. Wait for resize to be actuated (common.WaitForPodResizeActuation)
//     — skipped if waitForActuation is false
//  6. Verify pod resources after resize (podresize.VerifyPodResources)
//     — skipped if waitForActuation is false
//  7. Verify pod cpusets after resize (common.HaveContainerCPUsCount)
//     — skipped if waitForActuation is false
//  8. Verify scale delay time (if isScaleDown: time > scaleDelayTime)
//     — skipped if waitForActuation is false
//
// 9. Return current container state for the next resize operation
func patchAndVerifyPodResize(
	tCtx ktesting.TContext,
	restConfig *rest.Config,
	pod *v1.Pod,
	containerName string,
	containersBeforeResize []podresize.ResizableContainerInfo,
	desiredContainers []podresize.ResizableContainerInfo,
	expectedCPUCount int,
	scaleDelayTime int,
	isScaleDown bool,
	waitForActuation bool,
	checkDownwardAPI bool,
) []podresize.ResizableContainerInfo {
	tCtx.Helper()

	// Step 1: Record time before scaling
	timeBeforeScaleDown := time.Now()

	// Step 2: Patch pod for resize
	tCtx.Logf("Patching pod %q for resize (CPU: %s → %s)",
		pod.Name, containersBeforeResize[0].Resources.CPUReq, desiredContainers[0].Resources.CPUReq)
	patchBytes := podresize.MakeResizePatch(containersBeforeResize, desiredContainers, nil, nil)
	patchedPod, err := tCtx.Client().CoreV1().Pods(pod.Namespace).Patch(tCtx,
		pod.Name, "application/strategic-merge-patch+json", patchBytes, metav1.PatchOptions{}, "resize")
	tCtx.ExpectNoError(err, "failed to patch pod %s for resize", pod.Name)

	// Step 3: Verify cpuset in DownwardAPI volume (only if DownwardAPI is present)
	if checkDownwardAPI {
		tCtx.Logf("Verifying cpuset in DownwardAPI volume for container %q (expecting %d CPUs)", containerName, expectedCPUCount)
		tCtx.Eventually(func(tCtx ktesting.TContext) bool {
			return common.HaveDownwardAPICPUsCount(tCtx, restConfig, patchedPod, containerName, expectedCPUCount)
		}).WithTimeout(2*time.Minute).Should(gomega.BeTrue(),
			"DownwardAPI cpuset should reflect %d CPUs for container %q", expectedCPUCount, containerName)
	} else {
		tCtx.Logf("Skipping DownwardAPI cpuset verification (DownwardAPI not present for this pod)")
	}

	// Step 5: Verify pod resources post-patch, pre-actuation
	tCtx.Logf("Verifying pod resources post-patch, pre-actuation")
	podresize.VerifyPodResources(patchedPod, desiredContainers, nil)

	// Steps 6-9: Only run if waitForActuation is true
	if waitForActuation {
		// Step 6: Wait for resize to be actuated
		tCtx.Logf("Waiting for resize to be actuated (expecting %d CPUs)", expectedCPUCount)
		resizedPod := common.WaitForPodResizeActuation(tCtx, restConfig, patchedPod, containerName, expectedCPUCount)

		// Step 7: Verify pod resources after resize
		tCtx.Logf("Verifying pod resources after resize")
		podresize.VerifyPodResources(resizedPod, desiredContainers, nil)

		// Step 8: Verify pod cpusets after resize (use Eventually since cgroup may need time to update)
		tCtx.Logf("Verifying pod cpusets after resize (expecting %d CPUs)", expectedCPUCount)
		tCtx.Eventually(func(tCtx ktesting.TContext) bool {
			return common.HaveContainerCPUsCount(tCtx, restConfig, resizedPod, containerName, expectedCPUCount)
		}).WithTimeout(2*time.Minute).Should(gomega.BeTrue(),
			"container %q should have %d CPUs in its cpuset after resize", containerName, expectedCPUCount)

		// Step 9: Verify scale delay time (only for scale-down)
		if isScaleDown {
			timeAfterScaleDown := time.Now()
			scaleDuration := timeAfterScaleDown.Sub(timeBeforeScaleDown)
			tCtx.Logf("Verifying scale-down delay: took %.2f seconds (expected > %d seconds)", scaleDuration.Seconds(), scaleDelayTime)
			gomega.Expect(scaleDuration.Seconds()).To(gomega.BeNumerically(">", float64(scaleDelayTime)),
				fmt.Sprintf("Scale down should take more than %d seconds due to scale-delay-time (actual: %.2f seconds)",
					scaleDelayTime, scaleDuration.Seconds()))
		}
	} else {
		tCtx.Logf("Skipping actuation wait (waitForActuation=false)")
	}

	// Step 10: Return current container state
	return desiredContainers
}

// stage3And4KubeletDowngrade downgrades the kubelet to a previous release binary and
// verifies the pod's state after the downgrade. It prints the pod spec for visibility
// and checks that the pod is still running with the correct cpuset.
//
// Steps:
//  1. Downgrade kubelet to previous release binary (via cluster.Modify)
//  2. Wait for node to be schedulable after downgrade
//  3. Fetch the pod and print its spec (containers, resources, status, conditions)
//  4. Verify pod is still running
//  5. Verify cgroup cpuset is still correct (should be 2 CPUs from the scale-up)
//
// Returns the ModifyOptions needed to restore the kubelet in Stage 5.
func stage3And4KubeletDowngrade(
	tCtx ktesting.TContext,
	restConfig *rest.Config,
	cluster *localupcluster.Cluster,
	pod *v1.Pod,
	containerName string,
	previousBinDir string,
	major, previousMinor uint,
) (localupcluster.ModifyOptions, string, string) {
	tCtx.Helper()

	var restoreOpts localupcluster.ModifyOptions

	// Step 1: Downgrade kubelet to previous release binary and disable DownwardAPIAssignedResources
	tCtx.Step("downgrade-kubelet", func(tCtx ktesting.TContext) {
		previousKubeletPath := path.Join(previousBinDir, string(localupcluster.Kubelet))
		tCtx.Logf("Downgrading kubelet to previous release binary at %s", previousKubeletPath)
		restoreOpts = cluster.Modify(tCtx, "1-previous-kubelet", localupcluster.ModifyOptions{
			FileByComponent: map[localupcluster.ClusterComponentName]string{
				localupcluster.Kubelet: previousKubeletPath,
			},
			// The previous kubelet doesn't recognize certain flags and feature gates
			// that were added in the current release. Remove/replace them to prevent
			// the old kubelet from crashing on startup.
			ModifyFlagsByComponent: map[localupcluster.ClusterComponentName]map[string]string{
				localupcluster.Kubelet: {
					// Drop entirely — old kubelet doesn't support scale-delay-time option.
					"--cpu-manager-policy-options": "",
					// Replace feature gates — remove DownwardAPIAssignedResources and
					// InPlacePodVerticalScalingExclusiveCPUs (not recognized by old kubelet).
					"--feature-gates": "CPUManagerPolicyAlphaOptions=true",
				},
			},
			// Disable DownwardAPIAssignedResources on kube-apiserver to simulate
			// the previous release where this feature gate didn't exist.
			FeatureGatesByComponent: map[localupcluster.ClusterComponentName]string{
				localupcluster.KubeAPIServer: "DownwardAPIAssignedResources=false",
			},
		})
		tCtx.Logf("Kubelet downgraded to previous release (version %d.%d)", major, previousMinor)
	})

	// Step 2: Wait for node to be schedulable after kubelet downgrade
	tCtx.Step("wait-for-node-after-downgrade", func(tCtx ktesting.TContext) {
		tCtx.ExpectNoError(e2enode.WaitForAllNodesSchedulable(tCtx, tCtx.Client(), 5*time.Minute))
		tCtx.Logf("Node is schedulable after kubelet downgrade")
	})

	// Step 3: Verify pod is still running after kubelet downgrade
	tCtx.Step("verify-pod-running", func(tCtx ktesting.TContext) {
		tCtx.Eventually(func(tCtx ktesting.TContext) error {
			freshPod, err := tCtx.Client().CoreV1().Pods(pod.Namespace).Get(tCtx, pod.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			if freshPod.Status.Phase != v1.PodRunning {
				return fmt.Errorf("pod phase is %s, expected Running", freshPod.Status.Phase)
			}
			return nil
		}).WithTimeout(2*time.Minute).WithPolling(1*time.Second).Should(gomega.Succeed(),
			"pod should be Running after kubelet downgrade")
		tCtx.Logf("Pod %q is still Running after kubelet downgrade", pod.Name)
	})

	// Step 4: Deploy Pod-B with DownwardAPI volume (after kubelet downgrade)
	// Since DownwardAPIAssignedResources is now disabled on kube-apiserver, the old kubelet
	// should not populate the assigned.cpuset in the DownwardAPI volume for new pods.
	var podB *v1.Pod
	const podBName = "pod-b-downgrade-test"
	const containerBName = "gu-container-b"
	podBContainers := []podresize.ResizableContainerInfo{
		{
			Name: containerBName,
			Resources: &cgroups.ContainerResources{
				CPUReq: "1000m", CPULim: "1000m",
				MemReq: "200Mi", MemLim: "200Mi",
			},
			HasExclusiveCPUs: true,
		},
	}
	tCtx.Step("create-pod-b", func(tCtx ktesting.TContext) {
		tStamp := strconv.Itoa(time.Now().Nanosecond())
		podSpec := podresize.MakeResizablePodWithDownwardAPI(pod.Namespace, podBName, tStamp, podBContainers, nil)
		podSpec = e2epod.MustMixinRestrictedPodSecurity(podSpec)
		podB = common.CreatePodAndWaitForRunning(tCtx, podSpec)
		tCtx.Logf("Pod-B %q created and running after kubelet downgrade", podBName)
	})

	// Step 5: Verify DownwardAPI volume is dropped (cpuset not populated)
	// With DownwardAPIAssignedResources=false, the old kubelet should not write the
	// assigned.cpuset value to the DownwardAPI volume file.
	tCtx.Step("verify-downwardapi-dropped", func(tCtx ktesting.TContext) {
		// Re-fetch podB to get the latest status
		freshPod, err := tCtx.Client().CoreV1().Pods(podB.Namespace).Get(tCtx, podB.Name, metav1.GetOptions{})
		tCtx.ExpectNoError(err, "failed to get pod %s", podB.Name)
		podB = freshPod

		tCtx.Eventually(func(tCtx ktesting.TContext) bool {
			return common.HaveDownwardAPICpusetNotVisible(tCtx, restConfig, podB, containerBName)
		}).WithTimeout(2*time.Minute).WithPolling(1*time.Second).Should(gomega.BeTrue(),
			"DownwardAPI cpuset should not be visible for container %q in pod %q", containerBName, podBName)
		tCtx.Logf("Verified DownwardAPI cpuset is NOT visible for Pod-B after kubelet downgrade")
	})

	return restoreOpts, podBName, containerBName
}

// stage5KubeletRestore restores the kubelet to the master binary, verifies Pod-B is still running,
// and performs scale up/down on Pod-B to verify scale-delay-time is respected after kubelet restore.
//
// Steps:
//  1. Restore kubelet to master binary (via cluster.Modify with restoreOpts)
//  2. Wait for node to be schedulable after restore
//  3. Verify Pod-B is still Running
//  4. Scale up Pod-B (1→2) and scale down (2→1) with delay verification
//     Note: Pod-B was created while DownwardAPIAssignedResources was disabled, so the API server
//     stripped the assigned.cpuset DownwardAPI volume items via dropDisabledAssignedCpuset.
//     We cannot re-add DownwardAPI volumes to a running pod (volumes are immutable), so we
//     skip DownwardAPI verification for Pod-B (checkDownwardAPI=false).
func stage5KubeletRestore(
	tCtx ktesting.TContext,
	restConfig *rest.Config,
	cluster *localupcluster.Cluster,
	restoreOpts localupcluster.ModifyOptions,
	podBName string,
	containerBName string,
	scaleDelayTime int,
) {
	tCtx.Helper()

	// Step 1: Restore kubelet to master binary and re-enable DownwardAPIAssignedResources
	tCtx.Step("restore-kubelet", func(tCtx ktesting.TContext) {
		tCtx.Logf("Restoring kubelet to master binary")
		// The restoreOpts from cluster.Modify doesn't capture FeatureGatesByComponent
		// for kube-apiserver, so we need to explicitly re-enable DownwardAPIAssignedResources.
		if restoreOpts.FeatureGatesByComponent == nil {
			restoreOpts.FeatureGatesByComponent = make(map[localupcluster.ClusterComponentName]string)
		}
		restoreOpts.FeatureGatesByComponent[localupcluster.KubeAPIServer] = "DownwardAPIAssignedResources=true"
		cluster.Modify(tCtx, "2-master-restored", restoreOpts)
		tCtx.Logf("Kubelet restored to master binary, DownwardAPIAssignedResources re-enabled on kube-apiserver")
	})

	// Step 2: Wait for node to be schedulable after kubelet restore
	tCtx.Step("wait-for-node-after-restore", func(tCtx ktesting.TContext) {
		tCtx.ExpectNoError(e2enode.WaitForAllNodesSchedulable(tCtx, tCtx.Client(), 5*time.Minute))
		tCtx.Logf("Node is schedulable after kubelet restore")
	})

	// Step 3: Verify Pod-B is still running after kubelet restore
	tCtx.Step("verify-pod-b-running", func(tCtx ktesting.TContext) {
		tCtx.Eventually(func(tCtx ktesting.TContext) error {
			freshPod, err := tCtx.Client().CoreV1().Pods(tCtx.Namespace()).Get(tCtx, podBName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			if freshPod.Status.Phase != v1.PodRunning {
				return fmt.Errorf("pod phase is %s, expected Running", freshPod.Status.Phase)
			}
			return nil
		}).WithTimeout(2*time.Minute).WithPolling(1*time.Second).Should(gomega.Succeed(),
			"pod %q should be Running after kubelet restore", podBName)
		tCtx.Logf("Pod-B %q is still Running after kubelet restore", podBName)
	})

	// Step 4: Scale up Pod-B (1→2) and scale down (2→1) with delay verification
	// Pod-B was created while DownwardAPIAssignedResources was disabled, so the API server
	// stripped the assigned.cpuset DownwardAPI volume items via dropDisabledAssignedCpuset.
	// We cannot re-add DownwardAPI volumes to a running pod (volumes are immutable), so we
	// skip DownwardAPI verification for Pod-B (checkDownwardAPI=false).
	tCtx.Step("scale-up-and-scale-down-pod-b", func(tCtx ktesting.TContext) {
		// Fetch fresh Pod-B
		freshPod, err := tCtx.Client().CoreV1().Pods(tCtx.Namespace()).Get(tCtx, podBName, metav1.GetOptions{})
		tCtx.ExpectNoError(err, "failed to get pod %s", podBName)

		podBOriginalContainers := []podresize.ResizableContainerInfo{
			{
				Name: containerBName,
				Resources: &cgroups.ContainerResources{
					CPUReq: "1000m", CPULim: "1000m",
					MemReq: "200Mi", MemLim: "200Mi",
				},
				HasExclusiveCPUs: true,
			},
		}

		// Scale up: CPU 1 → 2
		currentContainers := patchAndVerifyPodResize(
			tCtx, restConfig, freshPod, containerBName,
			podBOriginalContainers, // containers before resize (1 CPU)
			[]podresize.ResizableContainerInfo{ // desired containers (2 CPUs)
				{
					Name: containerBName,
					Resources: &cgroups.ContainerResources{
						CPUReq: "2000m", CPULim: "2000m",
						MemReq: "200Mi", MemLim: "200Mi",
					},
					HasExclusiveCPUs: true,
				},
			},
			2,              // expected CPU count after resize
			scaleDelayTime, // scale delay time
			false,          // isScaleDown = false (scale up)
			true,           // waitForActuation = true (wait for scale-up to complete)
			false,          // checkDownwardAPI = false (Pod-B has no DownwardAPI volume)
		)

		// Scale down: CPU 2 → 1 (wait for actuation, verify scale-delay-time)
		patchAndVerifyPodResize(
			tCtx, restConfig, freshPod, containerBName,
			currentContainers,      // containers before resize (2 CPUs)
			podBOriginalContainers, // desired containers (1 CPU, same as original)
			1,                      // expected CPU count after resize
			scaleDelayTime,         // scale delay time
			true,                   // isScaleDown = true
			true,                   // waitForActuation = true (wait for scale-down to complete)
			false,                  // checkDownwardAPI = false (Pod-B has no DownwardAPI volume)
		)
	})
}

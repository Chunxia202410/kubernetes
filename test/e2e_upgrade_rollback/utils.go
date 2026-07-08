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

package e2eupgraderollback

import (
	"fmt"
	"os"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	"k8s.io/kubernetes/test/utils/client-go/ktesting"
)

// RepoRootDefault figures out whether an E2E suite is invoked in its directory (as in `go test ./test/e2e_upgrade_rollback`),
// directly in the root (as in `make test`), or somewhere deep inside
// the _output directory.
func RepoRootDefault() string {
	for i := range 10 {
		path := "." + strings.Repeat("/..", i)
		if _, err := os.Stat(path + "/test/e2e/framework"); err == nil {
			return path
		}
	}
	// Traditional default.
	return "../../"
}

// CurrentBinDir returns the environment variable name and the directory
// where the Kubernetes server binaries are located.
func CurrentBinDir() (envName, content string) {
	envName = "KUBERNETES_SERVER_BIN_DIR"
	content, _ = os.LookupEnv(envName)
	return
}


// CreatePodAndWaitForRunning creates a pod and waits for it to reach Running state.
// Returns the created pod object.
func CreatePodAndWaitForRunning(tCtx ktesting.TContext, pod *v1.Pod) *v1.Pod {
	tCtx.Helper()
	namespace := tCtx.Namespace()

	createdPod, err := tCtx.Client().CoreV1().Pods(namespace).Create(tCtx, pod, metav1.CreateOptions{})
	tCtx.ExpectNoError(err, "create pod %s", pod.Name)
	tCtx.Logf("Created pod %q in namespace %q", createdPod.Name, namespace)

	tCtx.ExpectNoError(e2epod.WaitForPodRunningInNamespace(tCtx, tCtx.Client(), createdPod),
		"wait for pod %q to be Running", createdPod.Name)
	tCtx.Logf("Pod %q is Running", createdPod.Name)

	return createdPod
}



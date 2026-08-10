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

package downwardapiassignedresources

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/component-helpers/nodedeclaredfeatures/types"
)

// Ensure the feature struct implements the unified Feature interface.
var _ types.Feature = &downwardAPIAssignedResourcesFeature{}

const (
	// DownwardAPIAssignedResources is both the feature name and the feature gate name.
	DownwardAPIAssignedResources = "DownwardAPIAssignedResources"
)

// Feature is the implementation for Downward API assigned resources support.
var Feature = &downwardAPIAssignedResourcesFeature{}

type downwardAPIAssignedResourcesFeature struct{}

// Name returns the feature's well-known name.
func (f *downwardAPIAssignedResourcesFeature) Name() string {
	return DownwardAPIAssignedResources
}

// Discover checks if the node supports Downward API assigned resources.
// This only requires the feature gate to be enabled.
// The actual assigned resource values (like cpuset) are determined at runtime
// and may be empty if the pod doesn't qualify, but the capability itself is available.
func (f *downwardAPIAssignedResourcesFeature) Discover(cfg *types.NodeConfiguration) bool {
	// Only check Feature Gate, independent of CPU Manager policy
	return cfg.FeatureGates.Enabled(DownwardAPIAssignedResources)
}

// Requirements returns the feature's dependencies.
func (f *downwardAPIAssignedResourcesFeature) Requirements() *types.FeatureRequirements {
	return &types.FeatureRequirements{
		EnabledFeatureGates: []string{DownwardAPIAssignedResources},
	}
}

// InferForScheduling checks if pod scheduling requires Downward API assigned resources.
// This returns true when the pod spec uses assigned.* resources (e.g., assigned.cpuset) in downward API volumes
// and the pod has not yet been scheduled (Status is nil).
//
// For pods that already have a Status (including running pods or pods that were previously scheduled),
// this returns false to allow them to continue running even if the feature is no longer available.
// This handles scenarios such as:
//   - Kubelet restarts with a different configuration (e.g., feature gate disabled)
//   - Node loses the capability to provide assigned resources
func (f *downwardAPIAssignedResourcesFeature) InferForScheduling(podInfo *types.PodInfo) bool {
	// If the pod already has a Status, it means the pod is already scheduled or running.
	// Skip the feature requirement check to allow the pod to continue.
	if podInfo.Status != nil {
		return false
	}

	if !podUsesAssignedResources(podInfo.Spec) {
		return false
	}
	return true
}

// InferForUpdate checks if a pod update requires Downward API assigned resources.
// Since pod volumes (including downward API volumes) are immutable, this always returns false.
func (f *downwardAPIAssignedResourcesFeature) InferForUpdate(oldPodInfo, newPodInfo *types.PodInfo) bool {
	return false
}

// MaxVersion returns nil (no upper version bound).
func (f *downwardAPIAssignedResourcesFeature) MaxVersion() *version.Version {
	return nil
}

// podUsesAssignedResources checks if the pod spec uses assigned.* resources in downward API volumes.
func podUsesAssignedResources(spec *v1.PodSpec) bool {
	if spec == nil {
		return false
	}

	// Check volumes for downward API
	for _, vol := range spec.Volumes {
		if vol.DownwardAPI != nil {
			for _, item := range vol.DownwardAPI.Items {
				if item.ResourceFieldRef != nil {
					if item.ResourceFieldRef.Resource == "assigned.cpuset" {
						return true
					}
				}
			}
		}
		// Check projected volumes for downward API
		if vol.Projected != nil {
			for _, source := range vol.Projected.Sources {
				if source.DownwardAPI != nil {
					for _, item := range source.DownwardAPI.Items {
						if item.ResourceFieldRef != nil &&
							item.ResourceFieldRef.Resource == "assigned.cpuset" {
							return true
						}
					}
				}
			}
		}
	}

	return false
}

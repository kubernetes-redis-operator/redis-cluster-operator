/*
Copyright 2022.

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

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RedisClusterSpec defines the desired state of RedisCluster
type RedisClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Masters specifies how many master nodes should be created in the Redis cluster.
	// +kubebuilder:validation:Required
	Masters int32 `json:"masters"`

	// ReplicasPerMaster specifies how many replicas should be attached to each master node in the Redis cluster.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=0
	ReplicasPerMaster int32 `json:"replicasPerMaster,omitempty"`

	// Config specifies the Redis config to be set in each redis node. The format matches the format of redis.conf, as a multiline yaml string
	Config string `json:"config,omitempty"`

	// PodSpec specifies the overrides or additions necessary for the redis pods. This allows you to override any pod settings necessary
	PodSpec v1.PodSpec `json:"podSpec,omitempty"`
}

// ClusterStateType is a string type for cluster state enum-like values
type ClusterStateType string

const (
	// ClusterStateNormal indicates that the cluster is in a normal operational state.
	ClusterStateNormal            ClusterStateType = "Normal"
	// ClusterStateScalingOut indicates that the cluster is currently scaling out (adding nodes).
	ClusterStateScalingOut        ClusterStateType = "ScalingOut"
	// ClusterStateScalingIn indicates that the cluster is currently scaling in (removing nodes).
	ClusterStateScalingIn         ClusterStateType = "ScalingIn"
	// ClusterStateIncreasingReplicas indicates that the cluster is currently increasing the number of replicas.
	ClusterStateIncreasingReplicas ClusterStateType = "IncreasingReplicas"
	// ClusterStateDecreasingReplicas indicates that the cluster is currently decreasing the number of replicas.
	ClusterStateDecreasingReplicas ClusterStateType = "DecreasingReplicas"
	// ClusterStateUpgradingVersion indicates that the cluster is currently upgrading its version.
	ClusterStateUpgradingVersion  ClusterStateType = "UpgradingVersion"
)

// RedisClusterStatus defines the observed state of RedisCluster
type RedisClusterStatus struct {
	// ClusterState represents the current state of the cluster.
	// Allowed values: Normal, ScalingOut, ScalingIn, IncreasingReplicas, DecreasingReplicas, UpgradingVersion
	ClusterState ClusterStateType `json:"clusterState,omitempty"`

	// Conditions represent the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// SetInOperationCondition sets or updates the InOperation condition in the status.
func (rcs *RedisClusterStatus) SetInOperationCondition(inOperation bool, reason, message string) {
	condition := metav1.Condition{
		Type:    "InOperation",
		Status:  metav1.ConditionFalse,
		Reason:  reason,
		Message: message,
	}
	if inOperation {
		condition.Status = metav1.ConditionTrue
	}
	meta.SetStatusCondition(&rcs.Conditions, condition)
}

// GetInOperationCondition returns the InOperation condition if present.
func (rcs *RedisClusterStatus) GetInOperationCondition() *metav1.Condition {
	for i := range rcs.Conditions {
		if rcs.Conditions[i].Type == "InOperation" {
			return &rcs.Conditions[i]
		}
	}
	return nil
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Masters",type="integer",JSONPath=".spec.masters"
// +kubebuilder:printcolumn:name="ReplicasPerMaster",type="integer",JSONPath=".spec.replicasPerMaster"
// +kubebuilder:printcolumn:name="ClusterState",type="string",JSONPath=".status.clusterState"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// RedisCluster is the Schema for the redisclusters API
type RedisCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RedisClusterSpec   `json:"spec,omitempty"`
	Status RedisClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RedisClusterList contains a list of RedisCluster
type RedisClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RedisCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RedisCluster{}, &RedisClusterList{})
}

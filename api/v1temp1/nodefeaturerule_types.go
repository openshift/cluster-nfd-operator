/*
Copyright 2021. The Kubernetes Authors.

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

package v1temp1

import (
	"github.com/openshift/cluster-nfd-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NodeFeatureRule resource specifies a configuration for feature-based
// customization of node objects, such as node labeling.
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=nfr,scope=Cluster
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
// +kubebuilder:resource:scope=Namespaced
type NodeFeatureRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   v1alpha1.NodeFeatureRuleSpec   `json:"spec,omitempty"`
	Status v1alpha1.NodeFeatureRuleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NodeFeatureRuleList contains a list of NodeFeatureRule objects.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NodeFeatureRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeFeatureRule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeFeatureRule{}, &NodeFeatureRuleList{})
}

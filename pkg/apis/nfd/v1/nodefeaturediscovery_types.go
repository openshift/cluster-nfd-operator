package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NodeFeatureDiscoverySpec defines the desired state of NodeFeatureDiscovery
// +k8s:openapi-gen=true
type NodeFeatureDiscoverySpec struct {
	Operand      OperandSpec `json:"operand"`
	WorkerConfig ConfigSpec  `json:"workerConfig"`
}

// OperandSpec describes configuration options for the operand
type OperandSpec struct {
	// +kubebuilder:validation:Pattern=[a-zA-Z0-9\.\-\/]+
	Namespace string `json:"namespace,omitempty"`

	// +kubebuilder:validation:Pattern=[a-zA-Z0-9\-]+
	Image string `json:"image,omitempty"`

	// Image pull policy
	// +kubebuilder:validation:Optional
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`
}

// ConfigSpec describes configuration options for the NFD worker
type ConfigSpec struct {
	// BinaryData holds the NFD configuration file
	ConfigData string `json:"configData"`
}

// NodeFeatureDiscoveryStatus defines the observed state of NodeFeatureDiscovery
// +k8s:openapi-gen=true
type NodeFeatureDiscoveryStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeFeatureDiscovery is the Schema for the nodefeaturediscoveries API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=nodefeaturediscoveries,scope=Namespaced
type NodeFeatureDiscovery struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeFeatureDiscoverySpec   `json:"spec,omitempty"`
	Status NodeFeatureDiscoveryStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeFeatureDiscoveryList contains a list of NodeFeatureDiscovery
type NodeFeatureDiscoveryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeFeatureDiscovery `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeFeatureDiscovery{}, &NodeFeatureDiscoveryList{})
}

// ImagePath returns a compiled full valid image string
func (o *OperandSpec) ImagePath() string {
	return o.Image
}

// ImagePolicy returns a valid corev1.PullPolicy from the string in the CR
func (o *OperandSpec) ImagePolicy(pullPolicy string) corev1.PullPolicy {
	switch corev1.PullPolicy(pullPolicy) {
	case corev1.PullAlways:
		return corev1.PullAlways
	case corev1.PullNever:
		return corev1.PullNever
	}
	return corev1.PullIfNotPresent
}

// Data returns a valid ConfigMap name
func (c *ConfigSpec) Data() string {
	return c.ConfigData
}

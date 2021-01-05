package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// StorageOSClusterSpec defines the desired state of StorageOSCluster
type StorageOSClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of StorageOSCluster. Edit StorageOSCluster_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// StorageOSClusterStatus defines the observed state of StorageOSCluster
type StorageOSClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// StorageOSCluster is the Schema for the storageosclusters API
type StorageOSCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StorageOSClusterSpec   `json:"spec,omitempty"`
	Status StorageOSClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// StorageOSClusterList contains a list of StorageOSCluster
type StorageOSClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StorageOSCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StorageOSCluster{}, &StorageOSClusterList{})
}

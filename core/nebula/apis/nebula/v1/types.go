package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// genclient:nonNamespaced

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
type NebulaCA struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   NebulaCASpec   `json:"spec"`
	Status NebulaCAStatus `json:"status,omitempty"`
}

type NebulaCASpec struct {
	CAName     string `json:"caName"`
	SecretName string `json:"secretName"`
}

type NebulaCAStatus struct {
	State   NebulaCAState `json:"state,omitempty"`
	Message string        `json:"message,omitempty"`
}

type NebulaCAState string

const (
	NebulaCAStateCreating NebulaCAState = "Creating"
	NebulaCAStateReady    NebulaCAState = "Ready"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NebulaCAList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []NebulaCA `json:"items"`
}

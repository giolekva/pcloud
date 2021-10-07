package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
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

// +genclient
// genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NebulaNode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   NebulaNodeSpec   `json:"spec"`
	Status NebulaNodeStatus `json:"status,omitempty"`
}

type NebulaNodeSpec struct {
	CAName     string `json:"caName"`
	NodeName   string `json:"nodeName"`
	IPCidr     string `json:"ipCidr"`
	SecretName string `json:"secretName"`
}

type NebulaNodeStatus struct {
	State   NebulaNodeState `json:"state,omitempty"`
	Message string          `json:"message,omitempty"`
}

type NebulaNodeState string

const (
	NebulaNodeStateCreating NebulaNodeState = "Creating"
	NebulaNodeStateReady    NebulaNodeState = "Ready"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NebulaNodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []NebulaNode `json:"items"`
}

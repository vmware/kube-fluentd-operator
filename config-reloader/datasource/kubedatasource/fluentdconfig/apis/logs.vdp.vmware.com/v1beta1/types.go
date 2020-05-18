package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FluentdConfig defines the CRD
type FluentdConfig struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec FluentdConfigSpec `json:"spec,omitempty"`
}

// FluentdConfigSpec implements the fluent.conf file as CRD
type FluentdConfigSpec struct {
	FluentConf string `json:"fluentconf,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FluentdConfigList is the mandatory plural type
type FluentdConfigList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []FluentdConfig `json:"items"`
}

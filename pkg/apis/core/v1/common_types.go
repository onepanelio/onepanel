package v1

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ParameterOption struct {
	Name  string
	Value string
}

type Parameter struct {
	wfv1.Parameter
	Type    string
	Options []*ParameterOption
}

type Arguments struct {
	Parameters []Parameter `json:"parameters" protobuf:"bytes,1,opt,name=parameters"`
}

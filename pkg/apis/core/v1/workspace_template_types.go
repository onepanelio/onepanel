package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type WorkspaceTemplateSpec struct {
	WorkspaceSpec
}

type WorkspaceTemplate struct {
	metav1.ObjectMeta
	Spec WorkspaceTemplateSpec
}

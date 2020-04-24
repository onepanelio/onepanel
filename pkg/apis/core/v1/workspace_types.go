package v1

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	networking "istio.io/api/networking/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Workspace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              WorkspaceSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

type WorkspaceSpec struct {
	Arguments  *Arguments                 `json:"arguments" protobuf:"bytes,1,opt,name=arguments"`
	Containers []corev1.Container         `json:"containers" protobuf:"bytes,3,opt,name=containers"`
	Ports      []corev1.ServicePort       `json:"ports" protobuf:"bytes,4,opt,name=ports"`
	Routes     []*networking.HTTPRoute    `json:"routes" protobuf:"bytes,5,opt,name=routes"`
	Workflow   *wfv1.WorkflowTemplateSpec `json:"workflow" protobuf:"bytes,6,opt,name=workflow"`
}

package v1

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ParameterOption struct {
	Name  string `json:"name" protobuf:"bytes,1,opt,name=name"`
	Value string `json:"value" protobuf:"bytes,2,opt,name=value"`
}

type Parameter struct {
	Name     string             `json:"name" protobuf:"bytes,1,opt,name=name"`
	Value    *string            `json:"value,omitempty" protobuf:"bytes,2,opt,name=value"`
	Type     string             `json:"type" protobuf:"bytes,3,opt,name=type"`
	Options  []*ParameterOption `json:"options,omitempty" protobuf:"bytes,4,opt,name=options"`
	Required bool               `json:"required,omitempty" protobuf:"bytes,5,opt,name=required"`
}

type Arguments struct {
	Parameters []Parameter `json:"parameters" protobuf:"bytes,1,opt,name=parameters"`
}

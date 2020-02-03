package model

type ListOptions struct {
	LabelSelector string `json:"labelSelector,omitempty" protobuf:"bytes,1,opt,name=labelSelector"`
	FieldSelector string `json:"fieldSelector,omitempty" protobuf:"bytes,2,opt,name=fieldSelector"`
}

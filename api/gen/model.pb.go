// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.14.0
// source: model.proto

package gen

import (
	proto "github.com/golang/protobuf/proto"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

type ModelIdentifier struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Namespace string `protobuf:"bytes,1,opt,name=namespace,proto3" json:"namespace,omitempty"`
	Name      string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
}

func (x *ModelIdentifier) Reset() {
	*x = ModelIdentifier{}
	if protoimpl.UnsafeEnabled {
		mi := &file_model_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ModelIdentifier) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ModelIdentifier) ProtoMessage() {}

func (x *ModelIdentifier) ProtoReflect() protoreflect.Message {
	mi := &file_model_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ModelIdentifier.ProtoReflect.Descriptor instead.
func (*ModelIdentifier) Descriptor() ([]byte, []int) {
	return file_model_proto_rawDescGZIP(), []int{0}
}

func (x *ModelIdentifier) GetNamespace() string {
	if x != nil {
		return x.Namespace
	}
	return ""
}

func (x *ModelIdentifier) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type NodeSelector struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Key   string `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value string `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *NodeSelector) Reset() {
	*x = NodeSelector{}
	if protoimpl.UnsafeEnabled {
		mi := &file_model_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NodeSelector) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NodeSelector) ProtoMessage() {}

func (x *NodeSelector) ProtoReflect() protoreflect.Message {
	mi := &file_model_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NodeSelector.ProtoReflect.Descriptor instead.
func (*NodeSelector) Descriptor() ([]byte, []int) {
	return file_model_proto_rawDescGZIP(), []int{1}
}

func (x *NodeSelector) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *NodeSelector) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

type ResourceLimits struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Cpu    string `protobuf:"bytes,1,opt,name=cpu,proto3" json:"cpu,omitempty"`
	Memory string `protobuf:"bytes,2,opt,name=memory,proto3" json:"memory,omitempty"`
}

func (x *ResourceLimits) Reset() {
	*x = ResourceLimits{}
	if protoimpl.UnsafeEnabled {
		mi := &file_model_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ResourceLimits) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ResourceLimits) ProtoMessage() {}

func (x *ResourceLimits) ProtoReflect() protoreflect.Message {
	mi := &file_model_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ResourceLimits.ProtoReflect.Descriptor instead.
func (*ResourceLimits) Descriptor() ([]byte, []int) {
	return file_model_proto_rawDescGZIP(), []int{2}
}

func (x *ResourceLimits) GetCpu() string {
	if x != nil {
		return x.Cpu
	}
	return ""
}

func (x *ResourceLimits) GetMemory() string {
	if x != nil {
		return x.Memory
	}
	return ""
}

type PredictorServer struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name           string          `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	RuntimeVersion string          `protobuf:"bytes,2,opt,name=runtimeVersion,proto3" json:"runtimeVersion,omitempty"`
	StorageUri     string          `protobuf:"bytes,3,opt,name=storageUri,proto3" json:"storageUri,omitempty"`
	Limits         *ResourceLimits `protobuf:"bytes,4,opt,name=limits,proto3" json:"limits,omitempty"`
}

func (x *PredictorServer) Reset() {
	*x = PredictorServer{}
	if protoimpl.UnsafeEnabled {
		mi := &file_model_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PredictorServer) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PredictorServer) ProtoMessage() {}

func (x *PredictorServer) ProtoReflect() protoreflect.Message {
	mi := &file_model_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PredictorServer.ProtoReflect.Descriptor instead.
func (*PredictorServer) Descriptor() ([]byte, []int) {
	return file_model_proto_rawDescGZIP(), []int{3}
}

func (x *PredictorServer) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *PredictorServer) GetRuntimeVersion() string {
	if x != nil {
		return x.RuntimeVersion
	}
	return ""
}

func (x *PredictorServer) GetStorageUri() string {
	if x != nil {
		return x.StorageUri
	}
	return ""
}

func (x *PredictorServer) GetLimits() *ResourceLimits {
	if x != nil {
		return x.Limits
	}
	return nil
}

type Env struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name  string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Value string `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *Env) Reset() {
	*x = Env{}
	if protoimpl.UnsafeEnabled {
		mi := &file_model_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Env) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Env) ProtoMessage() {}

func (x *Env) ProtoReflect() protoreflect.Message {
	mi := &file_model_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Env.ProtoReflect.Descriptor instead.
func (*Env) Descriptor() ([]byte, []int) {
	return file_model_proto_rawDescGZIP(), []int{4}
}

func (x *Env) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Env) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

type Container struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Image string `protobuf:"bytes,1,opt,name=image,proto3" json:"image,omitempty"`
	Name  string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Env   []*Env `protobuf:"bytes,3,rep,name=env,proto3" json:"env,omitempty"`
}

func (x *Container) Reset() {
	*x = Container{}
	if protoimpl.UnsafeEnabled {
		mi := &file_model_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Container) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Container) ProtoMessage() {}

func (x *Container) ProtoReflect() protoreflect.Message {
	mi := &file_model_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Container.ProtoReflect.Descriptor instead.
func (*Container) Descriptor() ([]byte, []int) {
	return file_model_proto_rawDescGZIP(), []int{5}
}

func (x *Container) GetImage() string {
	if x != nil {
		return x.Image
	}
	return ""
}

func (x *Container) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Container) GetEnv() []*Env {
	if x != nil {
		return x.Env
	}
	return nil
}

type Transformer struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Containers []*Container `protobuf:"bytes,1,rep,name=containers,proto3" json:"containers,omitempty"`
}

func (x *Transformer) Reset() {
	*x = Transformer{}
	if protoimpl.UnsafeEnabled {
		mi := &file_model_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Transformer) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Transformer) ProtoMessage() {}

func (x *Transformer) ProtoReflect() protoreflect.Message {
	mi := &file_model_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Transformer.ProtoReflect.Descriptor instead.
func (*Transformer) Descriptor() ([]byte, []int) {
	return file_model_proto_rawDescGZIP(), []int{6}
}

func (x *Transformer) GetContainers() []*Container {
	if x != nil {
		return x.Containers
	}
	return nil
}

type Predictor struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	NodeSelector *NodeSelector    `protobuf:"bytes,1,opt,name=nodeSelector,proto3" json:"nodeSelector,omitempty"`
	Server       *PredictorServer `protobuf:"bytes,2,opt,name=server,proto3" json:"server,omitempty"`
}

func (x *Predictor) Reset() {
	*x = Predictor{}
	if protoimpl.UnsafeEnabled {
		mi := &file_model_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Predictor) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Predictor) ProtoMessage() {}

func (x *Predictor) ProtoReflect() protoreflect.Message {
	mi := &file_model_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Predictor.ProtoReflect.Descriptor instead.
func (*Predictor) Descriptor() ([]byte, []int) {
	return file_model_proto_rawDescGZIP(), []int{7}
}

func (x *Predictor) GetNodeSelector() *NodeSelector {
	if x != nil {
		return x.NodeSelector
	}
	return nil
}

func (x *Predictor) GetServer() *PredictorServer {
	if x != nil {
		return x.Server
	}
	return nil
}

type DeployModelRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Namespace   string       `protobuf:"bytes,1,opt,name=namespace,proto3" json:"namespace,omitempty"`
	Name        string       `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Predictor   *Predictor   `protobuf:"bytes,3,opt,name=predictor,proto3" json:"predictor,omitempty"`
	Transformer *Transformer `protobuf:"bytes,4,opt,name=transformer,proto3" json:"transformer,omitempty"`
}

func (x *DeployModelRequest) Reset() {
	*x = DeployModelRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_model_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeployModelRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeployModelRequest) ProtoMessage() {}

func (x *DeployModelRequest) ProtoReflect() protoreflect.Message {
	mi := &file_model_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeployModelRequest.ProtoReflect.Descriptor instead.
func (*DeployModelRequest) Descriptor() ([]byte, []int) {
	return file_model_proto_rawDescGZIP(), []int{8}
}

func (x *DeployModelRequest) GetNamespace() string {
	if x != nil {
		return x.Namespace
	}
	return ""
}

func (x *DeployModelRequest) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *DeployModelRequest) GetPredictor() *Predictor {
	if x != nil {
		return x.Predictor
	}
	return nil
}

func (x *DeployModelRequest) GetTransformer() *Transformer {
	if x != nil {
		return x.Transformer
	}
	return nil
}

type DeployModelResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Status string `protobuf:"bytes,1,opt,name=status,proto3" json:"status,omitempty"`
}

func (x *DeployModelResponse) Reset() {
	*x = DeployModelResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_model_proto_msgTypes[9]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeployModelResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeployModelResponse) ProtoMessage() {}

func (x *DeployModelResponse) ProtoReflect() protoreflect.Message {
	mi := &file_model_proto_msgTypes[9]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeployModelResponse.ProtoReflect.Descriptor instead.
func (*DeployModelResponse) Descriptor() ([]byte, []int) {
	return file_model_proto_rawDescGZIP(), []int{9}
}

func (x *DeployModelResponse) GetStatus() string {
	if x != nil {
		return x.Status
	}
	return ""
}

type ModelCondition struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	LastTransitionTime string `protobuf:"bytes,1,opt,name=lastTransitionTime,proto3" json:"lastTransitionTime,omitempty"`
	Status             string `protobuf:"bytes,2,opt,name=status,proto3" json:"status,omitempty"`
	Type               string `protobuf:"bytes,3,opt,name=type,proto3" json:"type,omitempty"`
}

func (x *ModelCondition) Reset() {
	*x = ModelCondition{}
	if protoimpl.UnsafeEnabled {
		mi := &file_model_proto_msgTypes[10]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ModelCondition) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ModelCondition) ProtoMessage() {}

func (x *ModelCondition) ProtoReflect() protoreflect.Message {
	mi := &file_model_proto_msgTypes[10]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ModelCondition.ProtoReflect.Descriptor instead.
func (*ModelCondition) Descriptor() ([]byte, []int) {
	return file_model_proto_rawDescGZIP(), []int{10}
}

func (x *ModelCondition) GetLastTransitionTime() string {
	if x != nil {
		return x.LastTransitionTime
	}
	return ""
}

func (x *ModelCondition) GetStatus() string {
	if x != nil {
		return x.Status
	}
	return ""
}

func (x *ModelCondition) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

type ModelStatus struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Ready      bool              `protobuf:"varint,1,opt,name=ready,proto3" json:"ready,omitempty"`
	Conditions []*ModelCondition `protobuf:"bytes,2,rep,name=conditions,proto3" json:"conditions,omitempty"`
}

func (x *ModelStatus) Reset() {
	*x = ModelStatus{}
	if protoimpl.UnsafeEnabled {
		mi := &file_model_proto_msgTypes[11]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ModelStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ModelStatus) ProtoMessage() {}

func (x *ModelStatus) ProtoReflect() protoreflect.Message {
	mi := &file_model_proto_msgTypes[11]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ModelStatus.ProtoReflect.Descriptor instead.
func (*ModelStatus) Descriptor() ([]byte, []int) {
	return file_model_proto_rawDescGZIP(), []int{11}
}

func (x *ModelStatus) GetReady() bool {
	if x != nil {
		return x.Ready
	}
	return false
}

func (x *ModelStatus) GetConditions() []*ModelCondition {
	if x != nil {
		return x.Conditions
	}
	return nil
}

var File_model_proto protoreflect.FileDescriptor

var file_model_proto_rawDesc = []byte{
	0x0a, 0x0b, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x03, 0x61,
	0x70, 0x69, 0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61,
	0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x43, 0x0a,
	0x0f, 0x4d, 0x6f, 0x64, 0x65, 0x6c, 0x49, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x66, 0x69, 0x65, 0x72,
	0x12, 0x1c, 0x0a, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x12, 0x12,
	0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x22, 0x36, 0x0a, 0x0c, 0x4e, 0x6f, 0x64, 0x65, 0x53, 0x65, 0x6c, 0x65, 0x63, 0x74,
	0x6f, 0x72, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x22, 0x3a, 0x0a, 0x0e, 0x52, 0x65,
	0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x4c, 0x69, 0x6d, 0x69, 0x74, 0x73, 0x12, 0x10, 0x0a, 0x03,
	0x63, 0x70, 0x75, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x63, 0x70, 0x75, 0x12, 0x16,
	0x0a, 0x06, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06,
	0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79, 0x22, 0x9a, 0x01, 0x0a, 0x0f, 0x50, 0x72, 0x65, 0x64, 0x69,
	0x63, 0x74, 0x6f, 0x72, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x26,
	0x0a, 0x0e, 0x72, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0e, 0x72, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x56,
	0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x1e, 0x0a, 0x0a, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67,
	0x65, 0x55, 0x72, 0x69, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x73, 0x74, 0x6f, 0x72,
	0x61, 0x67, 0x65, 0x55, 0x72, 0x69, 0x12, 0x2b, 0x0a, 0x06, 0x6c, 0x69, 0x6d, 0x69, 0x74, 0x73,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x52, 0x65, 0x73,
	0x6f, 0x75, 0x72, 0x63, 0x65, 0x4c, 0x69, 0x6d, 0x69, 0x74, 0x73, 0x52, 0x06, 0x6c, 0x69, 0x6d,
	0x69, 0x74, 0x73, 0x22, 0x2f, 0x0a, 0x03, 0x45, 0x6e, 0x76, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x14,
	0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x22, 0x51, 0x0a, 0x09, 0x43, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65,
	0x72, 0x12, 0x14, 0x0a, 0x05, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x05, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x1a, 0x0a, 0x03, 0x65,
	0x6e, 0x76, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x08, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x45,
	0x6e, 0x76, 0x52, 0x03, 0x65, 0x6e, 0x76, 0x22, 0x3d, 0x0a, 0x0b, 0x54, 0x72, 0x61, 0x6e, 0x73,
	0x66, 0x6f, 0x72, 0x6d, 0x65, 0x72, 0x12, 0x2e, 0x0a, 0x0a, 0x63, 0x6f, 0x6e, 0x74, 0x61, 0x69,
	0x6e, 0x65, 0x72, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x61, 0x70, 0x69,
	0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x52, 0x0a, 0x63, 0x6f, 0x6e, 0x74,
	0x61, 0x69, 0x6e, 0x65, 0x72, 0x73, 0x22, 0x70, 0x0a, 0x09, 0x50, 0x72, 0x65, 0x64, 0x69, 0x63,
	0x74, 0x6f, 0x72, 0x12, 0x35, 0x0a, 0x0c, 0x6e, 0x6f, 0x64, 0x65, 0x53, 0x65, 0x6c, 0x65, 0x63,
	0x74, 0x6f, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x61, 0x70, 0x69, 0x2e,
	0x4e, 0x6f, 0x64, 0x65, 0x53, 0x65, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x52, 0x0c, 0x6e, 0x6f,
	0x64, 0x65, 0x53, 0x65, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x12, 0x2c, 0x0a, 0x06, 0x73, 0x65,
	0x72, 0x76, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x61, 0x70, 0x69,
	0x2e, 0x50, 0x72, 0x65, 0x64, 0x69, 0x63, 0x74, 0x6f, 0x72, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72,
	0x52, 0x06, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x22, 0xa8, 0x01, 0x0a, 0x12, 0x44, 0x65, 0x70,
	0x6c, 0x6f, 0x79, 0x4d, 0x6f, 0x64, 0x65, 0x6c, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x1c, 0x0a, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x12, 0x12, 0x0a,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d,
	0x65, 0x12, 0x2c, 0x0a, 0x09, 0x70, 0x72, 0x65, 0x64, 0x69, 0x63, 0x74, 0x6f, 0x72, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x50, 0x72, 0x65, 0x64, 0x69,
	0x63, 0x74, 0x6f, 0x72, 0x52, 0x09, 0x70, 0x72, 0x65, 0x64, 0x69, 0x63, 0x74, 0x6f, 0x72, 0x12,
	0x32, 0x0a, 0x0b, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x66, 0x6f, 0x72, 0x6d, 0x65, 0x72, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73,
	0x66, 0x6f, 0x72, 0x6d, 0x65, 0x72, 0x52, 0x0b, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x66, 0x6f, 0x72,
	0x6d, 0x65, 0x72, 0x22, 0x2d, 0x0a, 0x13, 0x44, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x4d, 0x6f, 0x64,
	0x65, 0x6c, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x74,
	0x61, 0x74, 0x75, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74,
	0x75, 0x73, 0x22, 0x6c, 0x0a, 0x0e, 0x4d, 0x6f, 0x64, 0x65, 0x6c, 0x43, 0x6f, 0x6e, 0x64, 0x69,
	0x74, 0x69, 0x6f, 0x6e, 0x12, 0x2e, 0x0a, 0x12, 0x6c, 0x61, 0x73, 0x74, 0x54, 0x72, 0x61, 0x6e,
	0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x69, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x12, 0x6c, 0x61, 0x73, 0x74, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e,
	0x54, 0x69, 0x6d, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x12, 0x0a, 0x04,
	0x74, 0x79, 0x70, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65,
	0x22, 0x58, 0x0a, 0x0b, 0x4d, 0x6f, 0x64, 0x65, 0x6c, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12,
	0x14, 0x0a, 0x05, 0x72, 0x65, 0x61, 0x64, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x05,
	0x72, 0x65, 0x61, 0x64, 0x79, 0x12, 0x33, 0x0a, 0x0a, 0x63, 0x6f, 0x6e, 0x64, 0x69, 0x74, 0x69,
	0x6f, 0x6e, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x61, 0x70, 0x69, 0x2e,
	0x4d, 0x6f, 0x64, 0x65, 0x6c, 0x43, 0x6f, 0x6e, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0a,
	0x63, 0x6f, 0x6e, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x32, 0xd4, 0x02, 0x0a, 0x0c, 0x4d,
	0x6f, 0x64, 0x65, 0x6c, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x6b, 0x0a, 0x0b, 0x44,
	0x65, 0x70, 0x6c, 0x6f, 0x79, 0x4d, 0x6f, 0x64, 0x65, 0x6c, 0x12, 0x17, 0x2e, 0x61, 0x70, 0x69,
	0x2e, 0x44, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x4d, 0x6f, 0x64, 0x65, 0x6c, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x2b, 0x82, 0xd3, 0xe4,
	0x93, 0x02, 0x25, 0x22, 0x20, 0x2f, 0x61, 0x70, 0x69, 0x73, 0x2f, 0x76, 0x31, 0x62, 0x65, 0x74,
	0x61, 0x31, 0x2f, 0x7b, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x7d, 0x2f, 0x6d,
	0x6f, 0x64, 0x65, 0x6c, 0x73, 0x3a, 0x01, 0x2a, 0x12, 0x69, 0x0a, 0x0e, 0x47, 0x65, 0x74, 0x4d,
	0x6f, 0x64, 0x65, 0x6c, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x14, 0x2e, 0x61, 0x70, 0x69,
	0x2e, 0x4d, 0x6f, 0x64, 0x65, 0x6c, 0x49, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x66, 0x69, 0x65, 0x72,
	0x1a, 0x10, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x4d, 0x6f, 0x64, 0x65, 0x6c, 0x53, 0x74, 0x61, 0x74,
	0x75, 0x73, 0x22, 0x2f, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x29, 0x12, 0x27, 0x2f, 0x61, 0x70, 0x69,
	0x73, 0x2f, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2f, 0x7b, 0x6e, 0x61, 0x6d, 0x65, 0x73,
	0x70, 0x61, 0x63, 0x65, 0x7d, 0x2f, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x73, 0x2f, 0x7b, 0x6e, 0x61,
	0x6d, 0x65, 0x7d, 0x12, 0x6c, 0x0a, 0x0b, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x4d, 0x6f, 0x64,
	0x65, 0x6c, 0x12, 0x14, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x4d, 0x6f, 0x64, 0x65, 0x6c, 0x49, 0x64,
	0x65, 0x6e, 0x74, 0x69, 0x66, 0x69, 0x65, 0x72, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79,
	0x22, 0x2f, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x29, 0x2a, 0x27, 0x2f, 0x61, 0x70, 0x69, 0x73, 0x2f,
	0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2f, 0x7b, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61,
	0x63, 0x65, 0x7d, 0x2f, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x73, 0x2f, 0x7b, 0x6e, 0x61, 0x6d, 0x65,
	0x7d, 0x42, 0x24, 0x5a, 0x22, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x6f, 0x6e, 0x65, 0x70, 0x61, 0x6e, 0x65, 0x6c, 0x69, 0x6f, 0x2f, 0x63, 0x6f, 0x72, 0x65, 0x2f,
	0x61, 0x70, 0x69, 0x2f, 0x67, 0x65, 0x6e, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_model_proto_rawDescOnce sync.Once
	file_model_proto_rawDescData = file_model_proto_rawDesc
)

func file_model_proto_rawDescGZIP() []byte {
	file_model_proto_rawDescOnce.Do(func() {
		file_model_proto_rawDescData = protoimpl.X.CompressGZIP(file_model_proto_rawDescData)
	})
	return file_model_proto_rawDescData
}

var file_model_proto_msgTypes = make([]protoimpl.MessageInfo, 12)
var file_model_proto_goTypes = []interface{}{
	(*ModelIdentifier)(nil),     // 0: api.ModelIdentifier
	(*NodeSelector)(nil),        // 1: api.NodeSelector
	(*ResourceLimits)(nil),      // 2: api.ResourceLimits
	(*PredictorServer)(nil),     // 3: api.PredictorServer
	(*Env)(nil),                 // 4: api.Env
	(*Container)(nil),           // 5: api.Container
	(*Transformer)(nil),         // 6: api.Transformer
	(*Predictor)(nil),           // 7: api.Predictor
	(*DeployModelRequest)(nil),  // 8: api.DeployModelRequest
	(*DeployModelResponse)(nil), // 9: api.DeployModelResponse
	(*ModelCondition)(nil),      // 10: api.ModelCondition
	(*ModelStatus)(nil),         // 11: api.ModelStatus
	(*emptypb.Empty)(nil),       // 12: google.protobuf.Empty
}
var file_model_proto_depIdxs = []int32{
	2,  // 0: api.PredictorServer.limits:type_name -> api.ResourceLimits
	4,  // 1: api.Container.env:type_name -> api.Env
	5,  // 2: api.Transformer.containers:type_name -> api.Container
	1,  // 3: api.Predictor.nodeSelector:type_name -> api.NodeSelector
	3,  // 4: api.Predictor.server:type_name -> api.PredictorServer
	7,  // 5: api.DeployModelRequest.predictor:type_name -> api.Predictor
	6,  // 6: api.DeployModelRequest.transformer:type_name -> api.Transformer
	10, // 7: api.ModelStatus.conditions:type_name -> api.ModelCondition
	8,  // 8: api.ModelService.DeployModel:input_type -> api.DeployModelRequest
	0,  // 9: api.ModelService.GetModelStatus:input_type -> api.ModelIdentifier
	0,  // 10: api.ModelService.DeleteModel:input_type -> api.ModelIdentifier
	12, // 11: api.ModelService.DeployModel:output_type -> google.protobuf.Empty
	11, // 12: api.ModelService.GetModelStatus:output_type -> api.ModelStatus
	12, // 13: api.ModelService.DeleteModel:output_type -> google.protobuf.Empty
	11, // [11:14] is the sub-list for method output_type
	8,  // [8:11] is the sub-list for method input_type
	8,  // [8:8] is the sub-list for extension type_name
	8,  // [8:8] is the sub-list for extension extendee
	0,  // [0:8] is the sub-list for field type_name
}

func init() { file_model_proto_init() }
func file_model_proto_init() {
	if File_model_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_model_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ModelIdentifier); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_model_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*NodeSelector); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_model_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ResourceLimits); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_model_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PredictorServer); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_model_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Env); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_model_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Container); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_model_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Transformer); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_model_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Predictor); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_model_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeployModelRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_model_proto_msgTypes[9].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeployModelResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_model_proto_msgTypes[10].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ModelCondition); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_model_proto_msgTypes[11].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ModelStatus); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_model_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   12,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_model_proto_goTypes,
		DependencyIndexes: file_model_proto_depIdxs,
		MessageInfos:      file_model_proto_msgTypes,
	}.Build()
	File_model_proto = out.File
	file_model_proto_rawDesc = nil
	file_model_proto_goTypes = nil
	file_model_proto_depIdxs = nil
}

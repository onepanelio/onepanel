// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package gen

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion7

// ServiceServiceClient is the client API for ServiceService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ServiceServiceClient interface {
	GetService(ctx context.Context, in *GetServiceRequest, opts ...grpc.CallOption) (*Service, error)
	ListServices(ctx context.Context, in *ListServicesRequest, opts ...grpc.CallOption) (*ListServicesResponse, error)
	HasService(ctx context.Context, in *HasServiceRequest, opts ...grpc.CallOption) (*HasServiceResponse, error)
}

type serviceServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewServiceServiceClient(cc grpc.ClientConnInterface) ServiceServiceClient {
	return &serviceServiceClient{cc}
}

func (c *serviceServiceClient) GetService(ctx context.Context, in *GetServiceRequest, opts ...grpc.CallOption) (*Service, error) {
	out := new(Service)
	err := c.cc.Invoke(ctx, "/api.ServiceService/GetService", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serviceServiceClient) ListServices(ctx context.Context, in *ListServicesRequest, opts ...grpc.CallOption) (*ListServicesResponse, error) {
	out := new(ListServicesResponse)
	err := c.cc.Invoke(ctx, "/api.ServiceService/ListServices", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serviceServiceClient) HasService(ctx context.Context, in *HasServiceRequest, opts ...grpc.CallOption) (*HasServiceResponse, error) {
	out := new(HasServiceResponse)
	err := c.cc.Invoke(ctx, "/api.ServiceService/HasService", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ServiceServiceServer is the server API for ServiceService service.
// All implementations must embed UnimplementedServiceServiceServer
// for forward compatibility
type ServiceServiceServer interface {
	GetService(context.Context, *GetServiceRequest) (*Service, error)
	ListServices(context.Context, *ListServicesRequest) (*ListServicesResponse, error)
	HasService(context.Context, *HasServiceRequest) (*HasServiceResponse, error)
	mustEmbedUnimplementedServiceServiceServer()
}

// UnimplementedServiceServiceServer must be embedded to have forward compatible implementations.
type UnimplementedServiceServiceServer struct {
}

func (UnimplementedServiceServiceServer) GetService(context.Context, *GetServiceRequest) (*Service, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetService not implemented")
}
func (UnimplementedServiceServiceServer) ListServices(context.Context, *ListServicesRequest) (*ListServicesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListServices not implemented")
}
func (UnimplementedServiceServiceServer) HasService(context.Context, *HasServiceRequest) (*HasServiceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method HasService not implemented")
}
func (UnimplementedServiceServiceServer) mustEmbedUnimplementedServiceServiceServer() {}

// UnsafeServiceServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ServiceServiceServer will
// result in compilation errors.
type UnsafeServiceServiceServer interface {
	mustEmbedUnimplementedServiceServiceServer()
}

func RegisterServiceServiceServer(s grpc.ServiceRegistrar, srv ServiceServiceServer) {
	s.RegisterService(&_ServiceService_serviceDesc, srv)
}

func _ServiceService_GetService_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetServiceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServiceServer).GetService(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.ServiceService/GetService",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServiceServer).GetService(ctx, req.(*GetServiceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ServiceService_ListServices_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListServicesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServiceServer).ListServices(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.ServiceService/ListServices",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServiceServer).ListServices(ctx, req.(*ListServicesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ServiceService_HasService_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(HasServiceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServiceServer).HasService(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.ServiceService/HasService",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServiceServer).HasService(ctx, req.(*HasServiceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _ServiceService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "api.ServiceService",
	HandlerType: (*ServiceServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetService",
			Handler:    _ServiceService_GetService_Handler,
		},
		{
			MethodName: "ListServices",
			Handler:    _ServiceService_ListServices_Handler,
		},
		{
			MethodName: "HasService",
			Handler:    _ServiceService_HasService_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "services.proto",
}

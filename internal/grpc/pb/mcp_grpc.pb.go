// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v6.31.1
// source: mcp.proto

package pb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	MCPManager_ListServers_FullMethodName   = "/mcp.MCPManager/ListServers"
	MCPManager_GetServer_FullMethodName     = "/mcp.MCPManager/GetServer"
	MCPManager_StartServer_FullMethodName   = "/mcp.MCPManager/StartServer"
	MCPManager_StopServer_FullMethodName    = "/mcp.MCPManager/StopServer"
	MCPManager_GetTools_FullMethodName      = "/mcp.MCPManager/GetTools"
	MCPManager_GetConfig_FullMethodName     = "/mcp.MCPManager/GetConfig"
	MCPManager_ReloadConfig_FullMethodName  = "/mcp.MCPManager/ReloadConfig"
	MCPManager_GetConfigPath_FullMethodName = "/mcp.MCPManager/GetConfigPath"
	MCPManager_Subscribe_FullMethodName     = "/mcp.MCPManager/Subscribe"
	MCPManager_Health_FullMethodName        = "/mcp.MCPManager/Health"
)

// MCPManagerClient is the client API for MCPManager service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type MCPManagerClient interface {
	// Basic server operations
	ListServers(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*ServerList, error)
	GetServer(ctx context.Context, in *ServerRequest, opts ...grpc.CallOption) (*Server, error)
	StartServer(ctx context.Context, in *ServerRequest, opts ...grpc.CallOption) (*Server, error)
	StopServer(ctx context.Context, in *ServerRequest, opts ...grpc.CallOption) (*Server, error)
	// Tool information
	GetTools(ctx context.Context, in *ServerRequest, opts ...grpc.CallOption) (*ToolList, error)
	// Configuration
	GetConfig(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Config, error)
	ReloadConfig(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*StatusResponse, error)
	GetConfigPath(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*PathResponse, error)
	// Real-time streaming
	Subscribe(ctx context.Context, in *SubscribeRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[Event], error)
	// Health check
	Health(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*HealthStatus, error)
}

type mCPManagerClient struct {
	cc grpc.ClientConnInterface
}

func NewMCPManagerClient(cc grpc.ClientConnInterface) MCPManagerClient {
	return &mCPManagerClient{cc}
}

func (c *mCPManagerClient) ListServers(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*ServerList, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ServerList)
	err := c.cc.Invoke(ctx, MCPManager_ListServers_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *mCPManagerClient) GetServer(ctx context.Context, in *ServerRequest, opts ...grpc.CallOption) (*Server, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Server)
	err := c.cc.Invoke(ctx, MCPManager_GetServer_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *mCPManagerClient) StartServer(ctx context.Context, in *ServerRequest, opts ...grpc.CallOption) (*Server, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Server)
	err := c.cc.Invoke(ctx, MCPManager_StartServer_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *mCPManagerClient) StopServer(ctx context.Context, in *ServerRequest, opts ...grpc.CallOption) (*Server, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Server)
	err := c.cc.Invoke(ctx, MCPManager_StopServer_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *mCPManagerClient) GetTools(ctx context.Context, in *ServerRequest, opts ...grpc.CallOption) (*ToolList, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ToolList)
	err := c.cc.Invoke(ctx, MCPManager_GetTools_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *mCPManagerClient) GetConfig(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Config, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Config)
	err := c.cc.Invoke(ctx, MCPManager_GetConfig_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *mCPManagerClient) ReloadConfig(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*StatusResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(StatusResponse)
	err := c.cc.Invoke(ctx, MCPManager_ReloadConfig_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *mCPManagerClient) GetConfigPath(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*PathResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(PathResponse)
	err := c.cc.Invoke(ctx, MCPManager_GetConfigPath_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *mCPManagerClient) Subscribe(ctx context.Context, in *SubscribeRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[Event], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &MCPManager_ServiceDesc.Streams[0], MCPManager_Subscribe_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[SubscribeRequest, Event]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type MCPManager_SubscribeClient = grpc.ServerStreamingClient[Event]

func (c *mCPManagerClient) Health(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*HealthStatus, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(HealthStatus)
	err := c.cc.Invoke(ctx, MCPManager_Health_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MCPManagerServer is the server API for MCPManager service.
// All implementations must embed UnimplementedMCPManagerServer
// for forward compatibility.
type MCPManagerServer interface {
	// Basic server operations
	ListServers(context.Context, *Empty) (*ServerList, error)
	GetServer(context.Context, *ServerRequest) (*Server, error)
	StartServer(context.Context, *ServerRequest) (*Server, error)
	StopServer(context.Context, *ServerRequest) (*Server, error)
	// Tool information
	GetTools(context.Context, *ServerRequest) (*ToolList, error)
	// Configuration
	GetConfig(context.Context, *Empty) (*Config, error)
	ReloadConfig(context.Context, *Empty) (*StatusResponse, error)
	GetConfigPath(context.Context, *Empty) (*PathResponse, error)
	// Real-time streaming
	Subscribe(*SubscribeRequest, grpc.ServerStreamingServer[Event]) error
	// Health check
	Health(context.Context, *Empty) (*HealthStatus, error)
	mustEmbedUnimplementedMCPManagerServer()
}

// UnimplementedMCPManagerServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedMCPManagerServer struct{}

func (UnimplementedMCPManagerServer) ListServers(context.Context, *Empty) (*ServerList, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListServers not implemented")
}
func (UnimplementedMCPManagerServer) GetServer(context.Context, *ServerRequest) (*Server, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetServer not implemented")
}
func (UnimplementedMCPManagerServer) StartServer(context.Context, *ServerRequest) (*Server, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StartServer not implemented")
}
func (UnimplementedMCPManagerServer) StopServer(context.Context, *ServerRequest) (*Server, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StopServer not implemented")
}
func (UnimplementedMCPManagerServer) GetTools(context.Context, *ServerRequest) (*ToolList, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTools not implemented")
}
func (UnimplementedMCPManagerServer) GetConfig(context.Context, *Empty) (*Config, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetConfig not implemented")
}
func (UnimplementedMCPManagerServer) ReloadConfig(context.Context, *Empty) (*StatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReloadConfig not implemented")
}
func (UnimplementedMCPManagerServer) GetConfigPath(context.Context, *Empty) (*PathResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetConfigPath not implemented")
}
func (UnimplementedMCPManagerServer) Subscribe(*SubscribeRequest, grpc.ServerStreamingServer[Event]) error {
	return status.Errorf(codes.Unimplemented, "method Subscribe not implemented")
}
func (UnimplementedMCPManagerServer) Health(context.Context, *Empty) (*HealthStatus, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Health not implemented")
}
func (UnimplementedMCPManagerServer) mustEmbedUnimplementedMCPManagerServer() {}
func (UnimplementedMCPManagerServer) testEmbeddedByValue()                    {}

// UnsafeMCPManagerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to MCPManagerServer will
// result in compilation errors.
type UnsafeMCPManagerServer interface {
	mustEmbedUnimplementedMCPManagerServer()
}

func RegisterMCPManagerServer(s grpc.ServiceRegistrar, srv MCPManagerServer) {
	// If the following call pancis, it indicates UnimplementedMCPManagerServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&MCPManager_ServiceDesc, srv)
}

func _MCPManager_ListServers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MCPManagerServer).ListServers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: MCPManager_ListServers_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MCPManagerServer).ListServers(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _MCPManager_GetServer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ServerRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MCPManagerServer).GetServer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: MCPManager_GetServer_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MCPManagerServer).GetServer(ctx, req.(*ServerRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _MCPManager_StartServer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ServerRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MCPManagerServer).StartServer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: MCPManager_StartServer_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MCPManagerServer).StartServer(ctx, req.(*ServerRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _MCPManager_StopServer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ServerRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MCPManagerServer).StopServer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: MCPManager_StopServer_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MCPManagerServer).StopServer(ctx, req.(*ServerRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _MCPManager_GetTools_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ServerRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MCPManagerServer).GetTools(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: MCPManager_GetTools_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MCPManagerServer).GetTools(ctx, req.(*ServerRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _MCPManager_GetConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MCPManagerServer).GetConfig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: MCPManager_GetConfig_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MCPManagerServer).GetConfig(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _MCPManager_ReloadConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MCPManagerServer).ReloadConfig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: MCPManager_ReloadConfig_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MCPManagerServer).ReloadConfig(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _MCPManager_GetConfigPath_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MCPManagerServer).GetConfigPath(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: MCPManager_GetConfigPath_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MCPManagerServer).GetConfigPath(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _MCPManager_Subscribe_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(SubscribeRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(MCPManagerServer).Subscribe(m, &grpc.GenericServerStream[SubscribeRequest, Event]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type MCPManager_SubscribeServer = grpc.ServerStreamingServer[Event]

func _MCPManager_Health_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MCPManagerServer).Health(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: MCPManager_Health_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MCPManagerServer).Health(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

// MCPManager_ServiceDesc is the grpc.ServiceDesc for MCPManager service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var MCPManager_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "mcp.MCPManager",
	HandlerType: (*MCPManagerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ListServers",
			Handler:    _MCPManager_ListServers_Handler,
		},
		{
			MethodName: "GetServer",
			Handler:    _MCPManager_GetServer_Handler,
		},
		{
			MethodName: "StartServer",
			Handler:    _MCPManager_StartServer_Handler,
		},
		{
			MethodName: "StopServer",
			Handler:    _MCPManager_StopServer_Handler,
		},
		{
			MethodName: "GetTools",
			Handler:    _MCPManager_GetTools_Handler,
		},
		{
			MethodName: "GetConfig",
			Handler:    _MCPManager_GetConfig_Handler,
		},
		{
			MethodName: "ReloadConfig",
			Handler:    _MCPManager_ReloadConfig_Handler,
		},
		{
			MethodName: "GetConfigPath",
			Handler:    _MCPManager_GetConfigPath_Handler,
		},
		{
			MethodName: "Health",
			Handler:    _MCPManager_Health_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Subscribe",
			Handler:       _MCPManager_Subscribe_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "mcp.proto",
}

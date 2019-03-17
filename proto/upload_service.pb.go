// Code generated by protoc-gen-go. DO NOT EDIT.
// source: upload_service.proto

package upload

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// UploadRequest is the request for Upload.
type UploadRequest struct {
	// File is the file to upload.
	File []byte `protobuf:"bytes,1,opt,name=file,proto3" json:"file,omitempty"`
	// File key to store in S3
	Key string `protobuf:"bytes,2,opt,name=key,proto3" json:"key,omitempty"`
	// The bucket we want to upload the file to.
	Bucket               string   `protobuf:"bytes,3,opt,name=bucket,proto3" json:"bucket,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *UploadRequest) Reset()         { *m = UploadRequest{} }
func (m *UploadRequest) String() string { return proto.CompactTextString(m) }
func (*UploadRequest) ProtoMessage()    {}
func (*UploadRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_upload_service_3354fad3bdaedc7d, []int{0}
}
func (m *UploadRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_UploadRequest.Unmarshal(m, b)
}
func (m *UploadRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_UploadRequest.Marshal(b, m, deterministic)
}
func (dst *UploadRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_UploadRequest.Merge(dst, src)
}
func (m *UploadRequest) XXX_Size() int {
	return xxx_messageInfo_UploadRequest.Size(m)
}
func (m *UploadRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_UploadRequest.DiscardUnknown(m)
}

var xxx_messageInfo_UploadRequest proto.InternalMessageInfo

func (m *UploadRequest) GetFile() []byte {
	if m != nil {
		return m.File
	}
	return nil
}

func (m *UploadRequest) GetKey() string {
	if m != nil {
		return m.Key
	}
	return ""
}

func (m *UploadRequest) GetBucket() string {
	if m != nil {
		return m.Bucket
	}
	return ""
}

// UploadResponse is the response for Upload.
type UploadResponse struct {
	// The location that the file uploaded to.
	Output               string   `protobuf:"bytes,1,opt,name=output,proto3" json:"output,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *UploadResponse) Reset()         { *m = UploadResponse{} }
func (m *UploadResponse) String() string { return proto.CompactTextString(m) }
func (*UploadResponse) ProtoMessage()    {}
func (*UploadResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_upload_service_3354fad3bdaedc7d, []int{1}
}
func (m *UploadResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_UploadResponse.Unmarshal(m, b)
}
func (m *UploadResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_UploadResponse.Marshal(b, m, deterministic)
}
func (dst *UploadResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_UploadResponse.Merge(dst, src)
}
func (m *UploadResponse) XXX_Size() int {
	return xxx_messageInfo_UploadResponse.Size(m)
}
func (m *UploadResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_UploadResponse.DiscardUnknown(m)
}

var xxx_messageInfo_UploadResponse proto.InternalMessageInfo

func (m *UploadResponse) GetOutput() string {
	if m != nil {
		return m.Output
	}
	return ""
}

func init() {
	proto.RegisterType((*UploadRequest)(nil), "upload.UploadRequest")
	proto.RegisterType((*UploadResponse)(nil), "upload.UploadResponse")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// UploadClient is the client API for Upload service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type UploadClient interface {
	// The function Uploads the given file.
	//
	// Returns the Location of the file as output.
	//
	// In case of an error the error is returned.
	Upload(ctx context.Context, in *UploadRequest, opts ...grpc.CallOption) (*UploadResponse, error)
}

type uploadClient struct {
	cc *grpc.ClientConn
}

func NewUploadClient(cc *grpc.ClientConn) UploadClient {
	return &uploadClient{cc}
}

func (c *uploadClient) Upload(ctx context.Context, in *UploadRequest, opts ...grpc.CallOption) (*UploadResponse, error) {
	out := new(UploadResponse)
	err := c.cc.Invoke(ctx, "/upload.Upload/Upload", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UploadServer is the server API for Upload service.
type UploadServer interface {
	// The function Uploads the given file.
	//
	// Returns the Location of the file as output.
	//
	// In case of an error the error is returned.
	Upload(context.Context, *UploadRequest) (*UploadResponse, error)
}

func RegisterUploadServer(s *grpc.Server, srv UploadServer) {
	s.RegisterService(&_Upload_serviceDesc, srv)
}

func _Upload_Upload_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UploadRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UploadServer).Upload(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/upload.Upload/Upload",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UploadServer).Upload(ctx, req.(*UploadRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Upload_serviceDesc = grpc.ServiceDesc{
	ServiceName: "upload.Upload",
	HandlerType: (*UploadServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Upload",
			Handler:    _Upload_Upload_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "upload_service.proto",
}

func init() {
	proto.RegisterFile("upload_service.proto", fileDescriptor_upload_service_3354fad3bdaedc7d)
}

var fileDescriptor_upload_service_3354fad3bdaedc7d = []byte{
	// 167 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xe2, 0x12, 0x29, 0x2d, 0xc8, 0xc9,
	0x4f, 0x4c, 0x89, 0x2f, 0x4e, 0x2d, 0x2a, 0xcb, 0x4c, 0x4e, 0xd5, 0x2b, 0x28, 0xca, 0x2f, 0xc9,
	0x17, 0x62, 0x83, 0x88, 0x2a, 0xf9, 0x72, 0xf1, 0x86, 0x82, 0x59, 0x41, 0xa9, 0x85, 0xa5, 0xa9,
	0xc5, 0x25, 0x42, 0x42, 0x5c, 0x2c, 0x69, 0x99, 0x39, 0xa9, 0x12, 0x8c, 0x0a, 0x8c, 0x1a, 0x3c,
	0x41, 0x60, 0xb6, 0x90, 0x00, 0x17, 0x73, 0x76, 0x6a, 0xa5, 0x04, 0x13, 0x50, 0x88, 0x33, 0x08,
	0xc4, 0x14, 0x12, 0xe3, 0x62, 0x4b, 0x2a, 0x4d, 0xce, 0x4e, 0x2d, 0x91, 0x60, 0x06, 0x0b, 0x42,
	0x79, 0x4a, 0x1a, 0x5c, 0x7c, 0x30, 0xe3, 0x8a, 0x0b, 0xf2, 0xf3, 0x8a, 0x53, 0x41, 0x2a, 0xf3,
	0x4b, 0x4b, 0x0a, 0x4a, 0x4b, 0xc0, 0x26, 0x02, 0x55, 0x42, 0x78, 0x46, 0xce, 0x5c, 0x6c, 0x10,
	0x95, 0x42, 0x96, 0x70, 0x96, 0xa8, 0x1e, 0xc4, 0x55, 0x7a, 0x28, 0x4e, 0x92, 0x12, 0x43, 0x17,
	0x86, 0x18, 0xad, 0xc4, 0x90, 0xc4, 0x06, 0xf6, 0x8c, 0x31, 0x20, 0x00, 0x00, 0xff, 0xff, 0x68,
	0x2a, 0x6c, 0xfa, 0xe4, 0x00, 0x00, 0x00,
}
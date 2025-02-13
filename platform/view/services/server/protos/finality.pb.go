// Code generated by protoc-gen-go. DO NOT EDIT.
// source: finality.proto

package protos

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type IsTxFinal struct {
	Txid                 string   `protobuf:"bytes,1,opt,name=txid,proto3" json:"txid,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *IsTxFinal) Reset()         { *m = IsTxFinal{} }
func (m *IsTxFinal) String() string { return proto.CompactTextString(m) }
func (*IsTxFinal) ProtoMessage()    {}
func (*IsTxFinal) Descriptor() ([]byte, []int) {
	return fileDescriptor_0144d353a635b215, []int{0}
}

func (m *IsTxFinal) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_IsTxFinal.Unmarshal(m, b)
}
func (m *IsTxFinal) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_IsTxFinal.Marshal(b, m, deterministic)
}
func (m *IsTxFinal) XXX_Merge(src proto.Message) {
	xxx_messageInfo_IsTxFinal.Merge(m, src)
}
func (m *IsTxFinal) XXX_Size() int {
	return xxx_messageInfo_IsTxFinal.Size(m)
}
func (m *IsTxFinal) XXX_DiscardUnknown() {
	xxx_messageInfo_IsTxFinal.DiscardUnknown(m)
}

var xxx_messageInfo_IsTxFinal proto.InternalMessageInfo

func (m *IsTxFinal) GetTxid() string {
	if m != nil {
		return m.Txid
	}
	return ""
}

type IsTxFinalResponse struct {
	Payload              []byte   `protobuf:"bytes,1,opt,name=payload,proto3" json:"payload,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *IsTxFinalResponse) Reset()         { *m = IsTxFinalResponse{} }
func (m *IsTxFinalResponse) String() string { return proto.CompactTextString(m) }
func (*IsTxFinalResponse) ProtoMessage()    {}
func (*IsTxFinalResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_0144d353a635b215, []int{1}
}

func (m *IsTxFinalResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_IsTxFinalResponse.Unmarshal(m, b)
}
func (m *IsTxFinalResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_IsTxFinalResponse.Marshal(b, m, deterministic)
}
func (m *IsTxFinalResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_IsTxFinalResponse.Merge(m, src)
}
func (m *IsTxFinalResponse) XXX_Size() int {
	return xxx_messageInfo_IsTxFinalResponse.Size(m)
}
func (m *IsTxFinalResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_IsTxFinalResponse.DiscardUnknown(m)
}

var xxx_messageInfo_IsTxFinalResponse proto.InternalMessageInfo

func (m *IsTxFinalResponse) GetPayload() []byte {
	if m != nil {
		return m.Payload
	}
	return nil
}

type IsHashFinal struct {
	Hash                 []byte   `protobuf:"bytes,1,opt,name=Hash,proto3" json:"Hash,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *IsHashFinal) Reset()         { *m = IsHashFinal{} }
func (m *IsHashFinal) String() string { return proto.CompactTextString(m) }
func (*IsHashFinal) ProtoMessage()    {}
func (*IsHashFinal) Descriptor() ([]byte, []int) {
	return fileDescriptor_0144d353a635b215, []int{2}
}

func (m *IsHashFinal) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_IsHashFinal.Unmarshal(m, b)
}
func (m *IsHashFinal) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_IsHashFinal.Marshal(b, m, deterministic)
}
func (m *IsHashFinal) XXX_Merge(src proto.Message) {
	xxx_messageInfo_IsHashFinal.Merge(m, src)
}
func (m *IsHashFinal) XXX_Size() int {
	return xxx_messageInfo_IsHashFinal.Size(m)
}
func (m *IsHashFinal) XXX_DiscardUnknown() {
	xxx_messageInfo_IsHashFinal.DiscardUnknown(m)
}

var xxx_messageInfo_IsHashFinal proto.InternalMessageInfo

func (m *IsHashFinal) GetHash() []byte {
	if m != nil {
		return m.Hash
	}
	return nil
}

type IsHashFinalResponse struct {
	Belief               bool     `protobuf:"varint,1,opt,name=belief,proto3" json:"belief,omitempty"`
	IsFinal              bool     `protobuf:"varint,2,opt,name=isFinal,proto3" json:"isFinal,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *IsHashFinalResponse) Reset()         { *m = IsHashFinalResponse{} }
func (m *IsHashFinalResponse) String() string { return proto.CompactTextString(m) }
func (*IsHashFinalResponse) ProtoMessage()    {}
func (*IsHashFinalResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_0144d353a635b215, []int{3}
}

func (m *IsHashFinalResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_IsHashFinalResponse.Unmarshal(m, b)
}
func (m *IsHashFinalResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_IsHashFinalResponse.Marshal(b, m, deterministic)
}
func (m *IsHashFinalResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_IsHashFinalResponse.Merge(m, src)
}
func (m *IsHashFinalResponse) XXX_Size() int {
	return xxx_messageInfo_IsHashFinalResponse.Size(m)
}
func (m *IsHashFinalResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_IsHashFinalResponse.DiscardUnknown(m)
}

var xxx_messageInfo_IsHashFinalResponse proto.InternalMessageInfo

func (m *IsHashFinalResponse) GetBelief() bool {
	if m != nil {
		return m.Belief
	}
	return false
}

func (m *IsHashFinalResponse) GetIsFinal() bool {
	if m != nil {
		return m.IsFinal
	}
	return false
}

func init() {
	proto.RegisterType((*IsTxFinal)(nil), "protos.IsTxFinal")
	proto.RegisterType((*IsTxFinalResponse)(nil), "protos.IsTxFinalResponse")
	proto.RegisterType((*IsHashFinal)(nil), "protos.IsHashFinal")
	proto.RegisterType((*IsHashFinalResponse)(nil), "protos.IsHashFinalResponse")
}

func init() { proto.RegisterFile("finality.proto", fileDescriptor_0144d353a635b215) }

var fileDescriptor_0144d353a635b215 = []byte{
	// 166 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x4b, 0xcb, 0xcc, 0x4b,
	0xcc, 0xc9, 0x2c, 0xa9, 0xd4, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x03, 0x53, 0xc5, 0x4a,
	0xf2, 0x5c, 0x9c, 0x9e, 0xc5, 0x21, 0x15, 0x6e, 0x20, 0x59, 0x21, 0x21, 0x2e, 0x96, 0x92, 0x8a,
	0xcc, 0x14, 0x09, 0x46, 0x05, 0x46, 0x0d, 0xce, 0x20, 0x30, 0x5b, 0x49, 0x97, 0x4b, 0x10, 0xae,
	0x20, 0x28, 0xb5, 0xb8, 0x20, 0x3f, 0xaf, 0x38, 0x55, 0x48, 0x82, 0x8b, 0xbd, 0x20, 0xb1, 0x32,
	0x27, 0x3f, 0x11, 0xa2, 0x96, 0x27, 0x08, 0xc6, 0x55, 0x52, 0xe4, 0xe2, 0xf6, 0x2c, 0xf6, 0x48,
	0x2c, 0xce, 0x80, 0x9b, 0x08, 0xe2, 0x40, 0x55, 0x81, 0xd9, 0x4a, 0xee, 0x5c, 0xc2, 0x48, 0x4a,
	0xe0, 0x66, 0x8a, 0x71, 0xb1, 0x25, 0xa5, 0xe6, 0x64, 0xa6, 0xa6, 0x81, 0x15, 0x73, 0x04, 0x41,
	0x79, 0x20, 0xbb, 0x32, 0x8b, 0xc1, 0x4a, 0x25, 0x98, 0xc0, 0x12, 0x30, 0xae, 0x13, 0x77, 0x14,
	0xd4, 0x17, 0x0d, 0x8c, 0x8c, 0x49, 0x10, 0xa6, 0x31, 0x20, 0x00, 0x00, 0xff, 0xff, 0x8b, 0xf6,
	0xe1, 0xdf, 0xe9, 0x00, 0x00, 0x00,
}

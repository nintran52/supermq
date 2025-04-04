// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v6.30.2
// source: domains/v1/domains.proto

package v1

import (
	v1 "github.com/absmach/supermq/api/grpc/common/v1"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type DeleteUserRes struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Deleted       bool                   `protobuf:"varint,1,opt,name=deleted,proto3" json:"deleted,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DeleteUserRes) Reset() {
	*x = DeleteUserRes{}
	mi := &file_domains_v1_domains_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DeleteUserRes) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteUserRes) ProtoMessage() {}

func (x *DeleteUserRes) ProtoReflect() protoreflect.Message {
	mi := &file_domains_v1_domains_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteUserRes.ProtoReflect.Descriptor instead.
func (*DeleteUserRes) Descriptor() ([]byte, []int) {
	return file_domains_v1_domains_proto_rawDescGZIP(), []int{0}
}

func (x *DeleteUserRes) GetDeleted() bool {
	if x != nil {
		return x.Deleted
	}
	return false
}

type DeleteUserReq struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DeleteUserReq) Reset() {
	*x = DeleteUserReq{}
	mi := &file_domains_v1_domains_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DeleteUserReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteUserReq) ProtoMessage() {}

func (x *DeleteUserReq) ProtoReflect() protoreflect.Message {
	mi := &file_domains_v1_domains_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteUserReq.ProtoReflect.Descriptor instead.
func (*DeleteUserReq) Descriptor() ([]byte, []int) {
	return file_domains_v1_domains_proto_rawDescGZIP(), []int{1}
}

func (x *DeleteUserReq) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

var File_domains_v1_domains_proto protoreflect.FileDescriptor

const file_domains_v1_domains_proto_rawDesc = "" +
	"\n" +
	"\x18domains/v1/domains.proto\x12\n" +
	"domains.v1\x1a\x16common/v1/common.proto\")\n" +
	"\rDeleteUserRes\x12\x18\n" +
	"\adeleted\x18\x01 \x01(\bR\adeleted\"\x1f\n" +
	"\rDeleteUserReq\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id2\xb1\x01\n" +
	"\x0eDomainsService\x12O\n" +
	"\x15DeleteUserFromDomains\x12\x19.domains.v1.DeleteUserReq\x1a\x19.domains.v1.DeleteUserRes\"\x00\x12N\n" +
	"\x0eRetrieveEntity\x12\x1c.common.v1.RetrieveEntityReq\x1a\x1c.common.v1.RetrieveEntityRes\"\x00B5Z3github.com/absmach/supermq/internal/grpc/domains/v1b\x06proto3"

var (
	file_domains_v1_domains_proto_rawDescOnce sync.Once
	file_domains_v1_domains_proto_rawDescData []byte
)

func file_domains_v1_domains_proto_rawDescGZIP() []byte {
	file_domains_v1_domains_proto_rawDescOnce.Do(func() {
		file_domains_v1_domains_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_domains_v1_domains_proto_rawDesc), len(file_domains_v1_domains_proto_rawDesc)))
	})
	return file_domains_v1_domains_proto_rawDescData
}

var file_domains_v1_domains_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_domains_v1_domains_proto_goTypes = []any{
	(*DeleteUserRes)(nil),        // 0: domains.v1.DeleteUserRes
	(*DeleteUserReq)(nil),        // 1: domains.v1.DeleteUserReq
	(*v1.RetrieveEntityReq)(nil), // 2: common.v1.RetrieveEntityReq
	(*v1.RetrieveEntityRes)(nil), // 3: common.v1.RetrieveEntityRes
}
var file_domains_v1_domains_proto_depIdxs = []int32{
	1, // 0: domains.v1.DomainsService.DeleteUserFromDomains:input_type -> domains.v1.DeleteUserReq
	2, // 1: domains.v1.DomainsService.RetrieveEntity:input_type -> common.v1.RetrieveEntityReq
	0, // 2: domains.v1.DomainsService.DeleteUserFromDomains:output_type -> domains.v1.DeleteUserRes
	3, // 3: domains.v1.DomainsService.RetrieveEntity:output_type -> common.v1.RetrieveEntityRes
	2, // [2:4] is the sub-list for method output_type
	0, // [0:2] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_domains_v1_domains_proto_init() }
func file_domains_v1_domains_proto_init() {
	if File_domains_v1_domains_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_domains_v1_domains_proto_rawDesc), len(file_domains_v1_domains_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_domains_v1_domains_proto_goTypes,
		DependencyIndexes: file_domains_v1_domains_proto_depIdxs,
		MessageInfos:      file_domains_v1_domains_proto_msgTypes,
	}.Build()
	File_domains_v1_domains_proto = out.File
	file_domains_v1_domains_proto_goTypes = nil
	file_domains_v1_domains_proto_depIdxs = nil
}

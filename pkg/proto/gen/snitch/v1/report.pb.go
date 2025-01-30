// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.4
// 	protoc        (unknown)
// source: snitch/v1/report.proto

package snitchpb

import (
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

type ReportRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	ReportText    string                 `protobuf:"bytes,1,opt,name=report_text,json=reportText,proto3" json:"report_text,omitempty"`
	ReporterId    int32                  `protobuf:"varint,2,opt,name=reporter_id,json=reporterId,proto3" json:"reporter_id,omitempty"`
	ReportedId    int32                  `protobuf:"varint,3,opt,name=reported_id,json=reportedId,proto3" json:"reported_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ReportRequest) Reset() {
	*x = ReportRequest{}
	mi := &file_snitch_v1_report_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ReportRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReportRequest) ProtoMessage() {}

func (x *ReportRequest) ProtoReflect() protoreflect.Message {
	mi := &file_snitch_v1_report_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReportRequest.ProtoReflect.Descriptor instead.
func (*ReportRequest) Descriptor() ([]byte, []int) {
	return file_snitch_v1_report_proto_rawDescGZIP(), []int{0}
}

func (x *ReportRequest) GetReportText() string {
	if x != nil {
		return x.ReportText
	}
	return ""
}

func (x *ReportRequest) GetReporterId() int32 {
	if x != nil {
		return x.ReporterId
	}
	return 0
}

func (x *ReportRequest) GetReportedId() int32 {
	if x != nil {
		return x.ReportedId
	}
	return 0
}

type ReportResponse struct {
	state            protoimpl.MessageState `protogen:"open.v1"`
	ReportSuccessful bool                   `protobuf:"varint,1,opt,name=report_successful,json=reportSuccessful,proto3" json:"report_successful,omitempty"`
	unknownFields    protoimpl.UnknownFields
	sizeCache        protoimpl.SizeCache
}

func (x *ReportResponse) Reset() {
	*x = ReportResponse{}
	mi := &file_snitch_v1_report_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ReportResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReportResponse) ProtoMessage() {}

func (x *ReportResponse) ProtoReflect() protoreflect.Message {
	mi := &file_snitch_v1_report_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReportResponse.ProtoReflect.Descriptor instead.
func (*ReportResponse) Descriptor() ([]byte, []int) {
	return file_snitch_v1_report_proto_rawDescGZIP(), []int{1}
}

func (x *ReportResponse) GetReportSuccessful() bool {
	if x != nil {
		return x.ReportSuccessful
	}
	return false
}

var File_snitch_v1_report_proto protoreflect.FileDescriptor

var file_snitch_v1_report_proto_rawDesc = string([]byte{
	0x0a, 0x16, 0x73, 0x6e, 0x69, 0x74, 0x63, 0x68, 0x2f, 0x76, 0x31, 0x2f, 0x72, 0x65, 0x70, 0x6f,
	0x72, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x09, 0x73, 0x6e, 0x69, 0x74, 0x63, 0x68,
	0x2e, 0x76, 0x31, 0x22, 0x72, 0x0a, 0x0d, 0x52, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x12, 0x1f, 0x0a, 0x0b, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x5f, 0x74,
	0x65, 0x78, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x72, 0x65, 0x70, 0x6f, 0x72,
	0x74, 0x54, 0x65, 0x78, 0x74, 0x12, 0x1f, 0x0a, 0x0b, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x65,
	0x72, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0a, 0x72, 0x65, 0x70, 0x6f,
	0x72, 0x74, 0x65, 0x72, 0x49, 0x64, 0x12, 0x1f, 0x0a, 0x0b, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74,
	0x65, 0x64, 0x5f, 0x69, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0a, 0x72, 0x65, 0x70,
	0x6f, 0x72, 0x74, 0x65, 0x64, 0x49, 0x64, 0x22, 0x3d, 0x0a, 0x0e, 0x52, 0x65, 0x70, 0x6f, 0x72,
	0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x2b, 0x0a, 0x11, 0x72, 0x65, 0x70,
	0x6f, 0x72, 0x74, 0x5f, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x66, 0x75, 0x6c, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x08, 0x52, 0x10, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x53, 0x75, 0x63, 0x63,
	0x65, 0x73, 0x73, 0x66, 0x75, 0x6c, 0x32, 0x50, 0x0a, 0x0d, 0x52, 0x65, 0x70, 0x6f, 0x72, 0x74,
	0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x3f, 0x0a, 0x06, 0x52, 0x65, 0x70, 0x6f, 0x72,
	0x74, 0x12, 0x18, 0x2e, 0x73, 0x6e, 0x69, 0x74, 0x63, 0x68, 0x2e, 0x76, 0x31, 0x2e, 0x52, 0x65,
	0x70, 0x6f, 0x72, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x19, 0x2e, 0x73, 0x6e,
	0x69, 0x74, 0x63, 0x68, 0x2e, 0x76, 0x31, 0x2e, 0x52, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x42, 0x1b, 0x5a, 0x19, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x73, 0x6e, 0x69, 0x74, 0x63, 0x68, 0x3b, 0x73, 0x6e, 0x69,
	0x74, 0x63, 0x68, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
})

var (
	file_snitch_v1_report_proto_rawDescOnce sync.Once
	file_snitch_v1_report_proto_rawDescData []byte
)

func file_snitch_v1_report_proto_rawDescGZIP() []byte {
	file_snitch_v1_report_proto_rawDescOnce.Do(func() {
		file_snitch_v1_report_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_snitch_v1_report_proto_rawDesc), len(file_snitch_v1_report_proto_rawDesc)))
	})
	return file_snitch_v1_report_proto_rawDescData
}

var file_snitch_v1_report_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_snitch_v1_report_proto_goTypes = []any{
	(*ReportRequest)(nil),  // 0: snitch.v1.ReportRequest
	(*ReportResponse)(nil), // 1: snitch.v1.ReportResponse
}
var file_snitch_v1_report_proto_depIdxs = []int32{
	0, // 0: snitch.v1.ReportService.Report:input_type -> snitch.v1.ReportRequest
	1, // 1: snitch.v1.ReportService.Report:output_type -> snitch.v1.ReportResponse
	1, // [1:2] is the sub-list for method output_type
	0, // [0:1] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_snitch_v1_report_proto_init() }
func file_snitch_v1_report_proto_init() {
	if File_snitch_v1_report_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_snitch_v1_report_proto_rawDesc), len(file_snitch_v1_report_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_snitch_v1_report_proto_goTypes,
		DependencyIndexes: file_snitch_v1_report_proto_depIdxs,
		MessageInfos:      file_snitch_v1_report_proto_msgTypes,
	}.Build()
	File_snitch_v1_report_proto = out.File
	file_snitch_v1_report_proto_goTypes = nil
	file_snitch_v1_report_proto_depIdxs = nil
}

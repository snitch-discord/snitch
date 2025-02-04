// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.4
// 	protoc        (unknown)
// source: snitch/v1/report.proto

package snitchv1

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

type Empty struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Empty) Reset() {
	*x = Empty{}
	mi := &file_snitch_v1_report_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Empty) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Empty) ProtoMessage() {}

func (x *Empty) ProtoReflect() protoreflect.Message {
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

// Deprecated: Use Empty.ProtoReflect.Descriptor instead.
func (*Empty) Descriptor() ([]byte, []int) {
	return file_snitch_v1_report_proto_rawDescGZIP(), []int{0}
}

type CreateReportRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	ReportText    string                 `protobuf:"bytes,1,opt,name=report_text,json=reportText,proto3" json:"report_text,omitempty"`
	ReporterId    int32                  `protobuf:"varint,2,opt,name=reporter_id,json=reporterId,proto3" json:"reporter_id,omitempty"`
	ReportedId    int32                  `protobuf:"varint,3,opt,name=reported_id,json=reportedId,proto3" json:"reported_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CreateReportRequest) Reset() {
	*x = CreateReportRequest{}
	mi := &file_snitch_v1_report_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateReportRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateReportRequest) ProtoMessage() {}

func (x *CreateReportRequest) ProtoReflect() protoreflect.Message {
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

// Deprecated: Use CreateReportRequest.ProtoReflect.Descriptor instead.
func (*CreateReportRequest) Descriptor() ([]byte, []int) {
	return file_snitch_v1_report_proto_rawDescGZIP(), []int{1}
}

func (x *CreateReportRequest) GetReportText() string {
	if x != nil {
		return x.ReportText
	}
	return ""
}

func (x *CreateReportRequest) GetReporterId() int32 {
	if x != nil {
		return x.ReporterId
	}
	return 0
}

func (x *CreateReportRequest) GetReportedId() int32 {
	if x != nil {
		return x.ReportedId
	}
	return 0
}

type CreateReportResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	ReportId      int32                  `protobuf:"varint,1,opt,name=report_id,json=reportId,proto3" json:"report_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CreateReportResponse) Reset() {
	*x = CreateReportResponse{}
	mi := &file_snitch_v1_report_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateReportResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateReportResponse) ProtoMessage() {}

func (x *CreateReportResponse) ProtoReflect() protoreflect.Message {
	mi := &file_snitch_v1_report_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateReportResponse.ProtoReflect.Descriptor instead.
func (*CreateReportResponse) Descriptor() ([]byte, []int) {
	return file_snitch_v1_report_proto_rawDescGZIP(), []int{2}
}

func (x *CreateReportResponse) GetReportId() int32 {
	if x != nil {
		return x.ReportId
	}
	return 0
}

type ListReportsResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Reports       []*CreateReportRequest `protobuf:"bytes,1,rep,name=reports,proto3" json:"reports,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ListReportsResponse) Reset() {
	*x = ListReportsResponse{}
	mi := &file_snitch_v1_report_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListReportsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListReportsResponse) ProtoMessage() {}

func (x *ListReportsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_snitch_v1_report_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListReportsResponse.ProtoReflect.Descriptor instead.
func (*ListReportsResponse) Descriptor() ([]byte, []int) {
	return file_snitch_v1_report_proto_rawDescGZIP(), []int{3}
}

func (x *ListReportsResponse) GetReports() []*CreateReportRequest {
	if x != nil {
		return x.Reports
	}
	return nil
}

var File_snitch_v1_report_proto protoreflect.FileDescriptor

var file_snitch_v1_report_proto_rawDesc = string([]byte{
	0x0a, 0x16, 0x73, 0x6e, 0x69, 0x74, 0x63, 0x68, 0x2f, 0x76, 0x31, 0x2f, 0x72, 0x65, 0x70, 0x6f,
	0x72, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x09, 0x73, 0x6e, 0x69, 0x74, 0x63, 0x68,
	0x2e, 0x76, 0x31, 0x22, 0x07, 0x0a, 0x05, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x78, 0x0a, 0x13,
	0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x52, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x1f, 0x0a, 0x0b, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x5f, 0x74, 0x65,
	0x78, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74,
	0x54, 0x65, 0x78, 0x74, 0x12, 0x1f, 0x0a, 0x0b, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x65, 0x72,
	0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0a, 0x72, 0x65, 0x70, 0x6f, 0x72,
	0x74, 0x65, 0x72, 0x49, 0x64, 0x12, 0x1f, 0x0a, 0x0b, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x65,
	0x64, 0x5f, 0x69, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0a, 0x72, 0x65, 0x70, 0x6f,
	0x72, 0x74, 0x65, 0x64, 0x49, 0x64, 0x22, 0x33, 0x0a, 0x14, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65,
	0x52, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x1b,
	0x0a, 0x09, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x05, 0x52, 0x08, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x49, 0x64, 0x22, 0x4f, 0x0a, 0x13, 0x4c,
	0x69, 0x73, 0x74, 0x52, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x12, 0x38, 0x0a, 0x07, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x73, 0x18, 0x01, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x1e, 0x2e, 0x73, 0x6e, 0x69, 0x74, 0x63, 0x68, 0x2e, 0x76, 0x31, 0x2e,
	0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x52, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x52, 0x07, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x73, 0x32, 0xa5, 0x01, 0x0a,
	0x0d, 0x52, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x51,
	0x0a, 0x0c, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x52, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x12, 0x1e,
	0x2e, 0x73, 0x6e, 0x69, 0x74, 0x63, 0x68, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74,
	0x65, 0x52, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1f,
	0x2e, 0x73, 0x6e, 0x69, 0x74, 0x63, 0x68, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74,
	0x65, 0x52, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22,
	0x00, 0x12, 0x41, 0x0a, 0x0b, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x73,
	0x12, 0x10, 0x2e, 0x73, 0x6e, 0x69, 0x74, 0x63, 0x68, 0x2e, 0x76, 0x31, 0x2e, 0x45, 0x6d, 0x70,
	0x74, 0x79, 0x1a, 0x1e, 0x2e, 0x73, 0x6e, 0x69, 0x74, 0x63, 0x68, 0x2e, 0x76, 0x31, 0x2e, 0x4c,
	0x69, 0x73, 0x74, 0x52, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x22, 0x00, 0x42, 0x29, 0x5a, 0x27, 0x73, 0x6e, 0x69, 0x74, 0x63, 0x68, 0x2f, 0x70,
	0x6b, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x73, 0x6e, 0x69,
	0x74, 0x63, 0x68, 0x2f, 0x76, 0x31, 0x3b, 0x73, 0x6e, 0x69, 0x74, 0x63, 0x68, 0x76, 0x31, 0x62,
	0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
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

var file_snitch_v1_report_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_snitch_v1_report_proto_goTypes = []any{
	(*Empty)(nil),                // 0: snitch.v1.Empty
	(*CreateReportRequest)(nil),  // 1: snitch.v1.CreateReportRequest
	(*CreateReportResponse)(nil), // 2: snitch.v1.CreateReportResponse
	(*ListReportsResponse)(nil),  // 3: snitch.v1.ListReportsResponse
}
var file_snitch_v1_report_proto_depIdxs = []int32{
	1, // 0: snitch.v1.ListReportsResponse.reports:type_name -> snitch.v1.CreateReportRequest
	1, // 1: snitch.v1.ReportService.CreateReport:input_type -> snitch.v1.CreateReportRequest
	0, // 2: snitch.v1.ReportService.ListReports:input_type -> snitch.v1.Empty
	2, // 3: snitch.v1.ReportService.CreateReport:output_type -> snitch.v1.CreateReportResponse
	3, // 4: snitch.v1.ReportService.ListReports:output_type -> snitch.v1.ListReportsResponse
	3, // [3:5] is the sub-list for method output_type
	1, // [1:3] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
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
			NumMessages:   4,
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

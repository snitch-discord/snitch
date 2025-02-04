// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: snitch/v1/report.proto

package snitchv1connect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	http "net/http"
	v1 "snitch/pkg/proto/gen/snitch/v1"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect.IsAtLeastVersion1_13_0

const (
	// ReportServiceName is the fully-qualified name of the ReportService service.
	ReportServiceName = "snitch.v1.ReportService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// ReportServiceCreateReportProcedure is the fully-qualified name of the ReportService's
	// CreateReport RPC.
	ReportServiceCreateReportProcedure = "/snitch.v1.ReportService/CreateReport"
	// ReportServiceListReportsProcedure is the fully-qualified name of the ReportService's ListReports
	// RPC.
	ReportServiceListReportsProcedure = "/snitch.v1.ReportService/ListReports"
)

// ReportServiceClient is a client for the snitch.v1.ReportService service.
type ReportServiceClient interface {
	CreateReport(context.Context, *connect.Request[v1.CreateReportRequest]) (*connect.Response[v1.CreateReportResponse], error)
	ListReports(context.Context, *connect.Request[v1.Empty]) (*connect.Response[v1.ListReportsResponse], error)
}

// NewReportServiceClient constructs a client for the snitch.v1.ReportService service. By default,
// it uses the Connect protocol with the binary Protobuf Codec, asks for gzipped responses, and
// sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC()
// or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewReportServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) ReportServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	reportServiceMethods := v1.File_snitch_v1_report_proto.Services().ByName("ReportService").Methods()
	return &reportServiceClient{
		createReport: connect.NewClient[v1.CreateReportRequest, v1.CreateReportResponse](
			httpClient,
			baseURL+ReportServiceCreateReportProcedure,
			connect.WithSchema(reportServiceMethods.ByName("CreateReport")),
			connect.WithClientOptions(opts...),
		),
		listReports: connect.NewClient[v1.Empty, v1.ListReportsResponse](
			httpClient,
			baseURL+ReportServiceListReportsProcedure,
			connect.WithSchema(reportServiceMethods.ByName("ListReports")),
			connect.WithClientOptions(opts...),
		),
	}
}

// reportServiceClient implements ReportServiceClient.
type reportServiceClient struct {
	createReport *connect.Client[v1.CreateReportRequest, v1.CreateReportResponse]
	listReports  *connect.Client[v1.Empty, v1.ListReportsResponse]
}

// CreateReport calls snitch.v1.ReportService.CreateReport.
func (c *reportServiceClient) CreateReport(ctx context.Context, req *connect.Request[v1.CreateReportRequest]) (*connect.Response[v1.CreateReportResponse], error) {
	return c.createReport.CallUnary(ctx, req)
}

// ListReports calls snitch.v1.ReportService.ListReports.
func (c *reportServiceClient) ListReports(ctx context.Context, req *connect.Request[v1.Empty]) (*connect.Response[v1.ListReportsResponse], error) {
	return c.listReports.CallUnary(ctx, req)
}

// ReportServiceHandler is an implementation of the snitch.v1.ReportService service.
type ReportServiceHandler interface {
	CreateReport(context.Context, *connect.Request[v1.CreateReportRequest]) (*connect.Response[v1.CreateReportResponse], error)
	ListReports(context.Context, *connect.Request[v1.Empty]) (*connect.Response[v1.ListReportsResponse], error)
}

// NewReportServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewReportServiceHandler(svc ReportServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	reportServiceMethods := v1.File_snitch_v1_report_proto.Services().ByName("ReportService").Methods()
	reportServiceCreateReportHandler := connect.NewUnaryHandler(
		ReportServiceCreateReportProcedure,
		svc.CreateReport,
		connect.WithSchema(reportServiceMethods.ByName("CreateReport")),
		connect.WithHandlerOptions(opts...),
	)
	reportServiceListReportsHandler := connect.NewUnaryHandler(
		ReportServiceListReportsProcedure,
		svc.ListReports,
		connect.WithSchema(reportServiceMethods.ByName("ListReports")),
		connect.WithHandlerOptions(opts...),
	)
	return "/snitch.v1.ReportService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case ReportServiceCreateReportProcedure:
			reportServiceCreateReportHandler.ServeHTTP(w, r)
		case ReportServiceListReportsProcedure:
			reportServiceListReportsHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedReportServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedReportServiceHandler struct{}

func (UnimplementedReportServiceHandler) CreateReport(context.Context, *connect.Request[v1.CreateReportRequest]) (*connect.Response[v1.CreateReportResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("snitch.v1.ReportService.CreateReport is not implemented"))
}

func (UnimplementedReportServiceHandler) ListReports(context.Context, *connect.Request[v1.Empty]) (*connect.Response[v1.ListReportsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("snitch.v1.ReportService.ListReports is not implemented"))
}

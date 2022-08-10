// Code generated by protoc-gen-grpc-gateway. DO NOT EDIT.
// source: api/profiles/profiles.proto

/*
Package profiles is a reverse proxy.

It translates gRPC into RESTful JSON APIs.
*/
package profiles

import (
	"context"
	"io"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/grpc-ecosystem/grpc-gateway/v2/utilities"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// Suppress "imported and not used" errors
var _ codes.Code
var _ io.Reader
var _ status.Status
var _ = runtime.String
var _ = utilities.NewDoubleArray
var _ = metadata.Join

func request_Profiles_GetProfiles_0(ctx context.Context, marshaler runtime.Marshaler, client ProfilesClient, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var protoReq GetProfilesRequest
	var metadata runtime.ServerMetadata

	msg, err := client.GetProfiles(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err

}

func local_request_Profiles_GetProfiles_0(ctx context.Context, marshaler runtime.Marshaler, server ProfilesServer, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var protoReq GetProfilesRequest
	var metadata runtime.ServerMetadata

	msg, err := server.GetProfiles(ctx, &protoReq)
	return msg, metadata, err

}

func request_Profiles_GetProfileValues_0(ctx context.Context, marshaler runtime.Marshaler, client ProfilesClient, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var protoReq GetProfileValuesRequest
	var metadata runtime.ServerMetadata

	var (
		val string
		ok  bool
		err error
		_   = err
	)

	val, ok = pathParams["profileName"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "profileName")
	}

	protoReq.ProfileName, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "profileName", err)
	}

	val, ok = pathParams["profileVersion"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "profileVersion")
	}

	protoReq.ProfileVersion, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "profileVersion", err)
	}

	msg, err := client.GetProfileValues(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err

}

func local_request_Profiles_GetProfileValues_0(ctx context.Context, marshaler runtime.Marshaler, server ProfilesServer, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var protoReq GetProfileValuesRequest
	var metadata runtime.ServerMetadata

	var (
		val string
		ok  bool
		err error
		_   = err
	)

	val, ok = pathParams["profileName"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "profileName")
	}

	protoReq.ProfileName, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "profileName", err)
	}

	val, ok = pathParams["profileVersion"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "profileVersion")
	}

	protoReq.ProfileVersion, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "profileVersion", err)
	}

	msg, err := server.GetProfileValues(ctx, &protoReq)
	return msg, metadata, err

}

// RegisterProfilesHandlerServer registers the http handlers for service Profiles to "mux".
// UnaryRPC     :call ProfilesServer directly.
// StreamingRPC :currently unsupported pending https://github.com/grpc/grpc-go/issues/906.
// Note that using this registration option will cause many gRPC library features to stop working. Consider using RegisterProfilesHandlerFromEndpoint instead.
func RegisterProfilesHandlerServer(ctx context.Context, mux *runtime.ServeMux, server ProfilesServer) error {

	mux.Handle("GET", pattern_Profiles_GetProfiles_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		var stream runtime.ServerTransportStream
		ctx = grpc.NewContextWithServerTransportStream(ctx, &stream)
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		rctx, err := runtime.AnnotateIncomingContext(ctx, mux, req, "/wego_profiles.v1.Profiles/GetProfiles", runtime.WithHTTPPathPattern("/v1/profiles"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := local_request_Profiles_GetProfiles_0(rctx, inboundMarshaler, server, req, pathParams)
		md.HeaderMD, md.TrailerMD = metadata.Join(md.HeaderMD, stream.Header()), metadata.Join(md.TrailerMD, stream.Trailer())
		ctx = runtime.NewServerMetadataContext(ctx, md)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}

		forward_Profiles_GetProfiles_0(ctx, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)

	})

	mux.Handle("GET", pattern_Profiles_GetProfileValues_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		var stream runtime.ServerTransportStream
		ctx = grpc.NewContextWithServerTransportStream(ctx, &stream)
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		rctx, err := runtime.AnnotateIncomingContext(ctx, mux, req, "/wego_profiles.v1.Profiles/GetProfileValues", runtime.WithHTTPPathPattern("/v1/profiles/{profileName}/{profileVersion}/values"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := local_request_Profiles_GetProfileValues_0(rctx, inboundMarshaler, server, req, pathParams)
		md.HeaderMD, md.TrailerMD = metadata.Join(md.HeaderMD, stream.Header()), metadata.Join(md.TrailerMD, stream.Trailer())
		ctx = runtime.NewServerMetadataContext(ctx, md)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}

		forward_Profiles_GetProfileValues_0(ctx, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)

	})

	return nil
}

// RegisterProfilesHandlerFromEndpoint is same as RegisterProfilesHandler but
// automatically dials to "endpoint" and closes the connection when "ctx" gets done.
func RegisterProfilesHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) (err error) {
	conn, err := grpc.Dial(endpoint, opts...)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if cerr := conn.Close(); cerr != nil {
				grpclog.Infof("Failed to close conn to %s: %v", endpoint, cerr)
			}
			return
		}
		go func() {
			<-ctx.Done()
			if cerr := conn.Close(); cerr != nil {
				grpclog.Infof("Failed to close conn to %s: %v", endpoint, cerr)
			}
		}()
	}()

	return RegisterProfilesHandler(ctx, mux, conn)
}

// RegisterProfilesHandler registers the http handlers for service Profiles to "mux".
// The handlers forward requests to the grpc endpoint over "conn".
func RegisterProfilesHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return RegisterProfilesHandlerClient(ctx, mux, NewProfilesClient(conn))
}

// RegisterProfilesHandlerClient registers the http handlers for service Profiles
// to "mux". The handlers forward requests to the grpc endpoint over the given implementation of "ProfilesClient".
// Note: the gRPC framework executes interceptors within the gRPC handler. If the passed in "ProfilesClient"
// doesn't go through the normal gRPC flow (creating a gRPC client etc.) then it will be up to the passed in
// "ProfilesClient" to call the correct interceptors.
func RegisterProfilesHandlerClient(ctx context.Context, mux *runtime.ServeMux, client ProfilesClient) error {

	mux.Handle("GET", pattern_Profiles_GetProfiles_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		rctx, err := runtime.AnnotateContext(ctx, mux, req, "/wego_profiles.v1.Profiles/GetProfiles", runtime.WithHTTPPathPattern("/v1/profiles"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := request_Profiles_GetProfiles_0(rctx, inboundMarshaler, client, req, pathParams)
		ctx = runtime.NewServerMetadataContext(ctx, md)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}

		forward_Profiles_GetProfiles_0(ctx, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)

	})

	mux.Handle("GET", pattern_Profiles_GetProfileValues_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		rctx, err := runtime.AnnotateContext(ctx, mux, req, "/wego_profiles.v1.Profiles/GetProfileValues", runtime.WithHTTPPathPattern("/v1/profiles/{profileName}/{profileVersion}/values"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := request_Profiles_GetProfileValues_0(rctx, inboundMarshaler, client, req, pathParams)
		ctx = runtime.NewServerMetadataContext(ctx, md)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}

		forward_Profiles_GetProfileValues_0(ctx, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)

	})

	return nil
}

var (
	pattern_Profiles_GetProfiles_0 = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1}, []string{"v1", "profiles"}, ""))

	pattern_Profiles_GetProfileValues_0 = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1, 1, 0, 4, 1, 5, 2, 1, 0, 4, 1, 5, 3, 2, 4}, []string{"v1", "profiles", "profileName", "profileVersion", "values"}, ""))
)

var (
	forward_Profiles_GetProfiles_0 = runtime.ForwardResponseMessage

	forward_Profiles_GetProfileValues_0 = runtime.ForwardResponseMessage
)

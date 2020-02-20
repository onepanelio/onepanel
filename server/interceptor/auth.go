package interceptor

import (
	"context"
	"net/http"
	"strings"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	v1 "github.com/onepanelio/core/pkg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func getBearerToken(ctx context.Context) (*string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, false
	}

	prefix := "Bearer "
	for _, t := range md.Get("authorization") {
		if !strings.HasPrefix(t, prefix) {
			return nil, false
		}
		t = strings.ReplaceAll(t, prefix, "")
		return &t, true
	}

	for _, c := range md.Get("grpcgateway-cookie") {
		header := http.Header{}
		header.Add("Cookie", c)
		req := &http.Request{
			Header: header,
		}
		t, _ := req.Cookie("auth-token")
		if t != nil {
			return &t.Value, true
		}
	}

	return nil, false
}

func getClient(ctx context.Context, kubeConfig *v1.Config, db *v1.DB) (context.Context, error) {
	bearerToken, ok := getBearerToken(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, `Missing or invalid "authorization" header.`)
	}

	kubeConfig.BearerToken = *bearerToken
	client, err := v1.NewClient(kubeConfig, db)
	if err != nil {
		return nil, err
	}

	return context.WithValue(ctx, "kubeClient", client), nil
}

func AuthUnaryInterceptor(kubeConfig *v1.Config, db *v1.DB) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		ctx, err = getClient(ctx, kubeConfig, db)
		if err != nil {
			return
		}

		return handler(ctx, req)
	}
}

func AuthStreamingInterceptor(kubeConfig *v1.Config, db *v1.DB) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		ctx, err := getClient(ss.Context(), kubeConfig, db)
		if err != nil {
			return
		}
		wrapped := grpc_middleware.WrapServerStream(ss)
		wrapped.WrappedContext = ctx

		return handler(srv, wrapped)
	}
}

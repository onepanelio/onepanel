package interceptor

import (
	"context"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	v1 "github.com/onepanelio/core/pkg"
	"google.golang.org/grpc"
)

func getClient(ctx context.Context, kubeConfig *v1.Config, db *v1.DB) (context.Context, error) {
	kubeConfig.BearerToken = ""
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

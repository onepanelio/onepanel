package auth

import (
	"context"
	"errors"
	"fmt"
	"github.com/onepanelio/core/api"
	"net/http"
	"strings"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	v1 "github.com/onepanelio/core/pkg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	authorizationv1 "k8s.io/api/authorization/v1"
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

func IsAuthorized(c *v1.Client, namespace, verb, group, resource, name string) (allowed bool, err error) {
	review, err := c.AuthorizationV1().SelfSubjectAccessReviews().Create(&authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: namespace,
				Verb:      verb,
				Group:     group,
				Resource:  resource,
				Name:      name,
			},
		},
	})
	if err != nil {
		return false, status.Error(codes.PermissionDenied, "Permission denied.")
	}
	allowed = review.Status.Allowed

	return
}

func UnaryInterceptor(kubeConfig *v1.Config, db *v1.DB) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		if info.FullMethod == "/api.AuthService/IsValidToken" {
			md, ok := metadata.FromIncomingContext(ctx)
			if !ok {
				return resp, errors.New("unable to get metadata from incoming context")
			}

			tokenRequest, ok := req.(*api.IsValidTokenRequest)
			if !ok {
				return resp, errors.New("IsValidToken does not have correct request type")
			}

			md.Set("authorization", tokenRequest.Token.Token)

			ctx, err = getClient(ctx, kubeConfig, db)
			if err != nil {
				ctx = nil
			}

			return handler(ctx, req)
		}

		// if you don't need the token,
		if info.FullMethod == "/api.AuthService/IsWorkspaceAuthenticated" {
			workspaceRequest, ok := req.(api.IsWorkspaceAuthenticatedRequest)
			if !ok {
				ctx = nil
				return handler(ctx, req)
			}

			//todo ignore fqdn
			//expected format: https://nginx-0--default.test-0.onepanel.site/
			fmt.Printf("%+v", workspaceRequest)
			if workspaceRequest.OriginalUri == "workspacerelated" {
				//getClient, for kube, will require token auth
				//set ctx
			}

			return handler(ctx, req)
		}

		// This guy checks for the token
		ctx, err = getClient(ctx, kubeConfig, db)
		if err != nil {
			return
		}

		return handler(ctx, req)
	}
}

func StreamingInterceptor(kubeConfig *v1.Config, db *v1.DB) grpc.StreamServerInterceptor {
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

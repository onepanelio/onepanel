package server

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/onepanelio/core/api"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/server/auth"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthServer struct{}

func (a *AuthServer) IsWorkspaceAuthenticated(ctx context.Context, request *api.IsWorkspaceAuthenticatedRequest) (*empty.Empty, error) {
	fmt.Printf("%+v\n", request)
	return &empty.Empty{}, nil
}

func NewAuthServer() *AuthServer {
	return &AuthServer{}
}

func (a *AuthServer) IsValidToken(ctx context.Context, req *api.IsValidTokenRequest) (*empty.Empty, error) {
	if ctx == nil {
		return nil, status.Error(codes.Unauthenticated, "Unauthenticated.")
	}

	client := ctx.Value("kubeClient").(*v1.Client)

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		if err.Error() == "Unauthorized" {
			return nil, status.Error(codes.Unauthenticated, "Unauthenticated.")
		}
		return nil, err
	}
	if len(namespaces) == 0 {
		return nil, errors.New("No namespaces for onepanel setup.")
	}
	namespace := namespaces[0]

	allowed, err := auth.IsAuthorized(client, "", "get", "", "namespaces", namespace.Name)
	if err != nil {
		return nil, err
	}

	if !allowed {
		return nil, status.Error(codes.Unauthenticated, "Unauthenticated.")
	}

	return &empty.Empty{}, nil
}

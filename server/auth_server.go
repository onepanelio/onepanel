package server

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/onepanelio/core/api"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/server/auth"
	"github.com/pkg/errors"
)

type AuthServer struct{}

func NewAuthServer() *AuthServer {
	return &AuthServer{}
}

func (a *AuthServer) IsValidToken(ctx context.Context, req *empty.Empty) (*api.IsValidTokenResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		if err.Error() == "Unauthorized" {
			return &api.IsValidTokenResponse{
				Valid: false,
			}, nil
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
		return &api.IsValidTokenResponse{
			Valid: false,
		}, nil
	}

	return &api.IsValidTokenResponse{
		Valid: true,
	}, nil
}

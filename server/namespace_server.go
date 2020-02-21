package server

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/onepanelio/core/api"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/server/auth"
)

type NamespaceServer struct{}

func NewNamespaceServer() *NamespaceServer {
	return &NamespaceServer{}
}

func apiNamespace(ns *v1.Namespace) (namespace *api.Namespace) {
	namespace = &api.Namespace{
		Name: ns.Name,
	}

	return
}

func (s *NamespaceServer) ListNamespaces(ctx context.Context, empty *empty.Empty) (*api.ListNamespacesResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, "", "list", "", "namespaces", "")
	if err != nil || !allowed {
		return nil, err
	}

	namespaces, err := client.ListNamespaces()
	if err != nil {
		return nil, err
	}

	apiNamespaces := []*api.Namespace{}
	for _, ns := range namespaces {
		apiNamespaces = append(apiNamespaces, apiNamespace(ns))
	}

	return &api.ListNamespacesResponse{
		Count:      int32(len(apiNamespaces)),
		Namespaces: apiNamespaces,
	}, nil
}

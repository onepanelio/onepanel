package server

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/onepanelio/core/api"
	"github.com/onepanelio/core/manager"
	"github.com/onepanelio/core/model"
	"github.com/onepanelio/core/util"
	"google.golang.org/grpc/codes"
)

type NamespaceServer struct {
	resourceManager *manager.ResourceManager
}

func NewNamespaceServer(resourceManager *manager.ResourceManager) *NamespaceServer {
	return &NamespaceServer{resourceManager: resourceManager}
}

func apiNamespace(ns *model.Namespace) (namespace *api.Namespace) {
	namespace = &api.Namespace{
		Name: ns.Name,
	}

	return
}

func (s *NamespaceServer) ListNamespaces(ctx context.Context, empty *empty.Empty) (*api.ListNamespacesResponse, error) {
	namespaces, err := s.resourceManager.ListNamespaces()
	if err != nil {
		return nil, util.NewUserError(codes.Unknown, "Unknown error.")
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

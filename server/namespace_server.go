package server

import (
	"context"
	"errors"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/onepanelio/core/api"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/util"
	"google.golang.org/grpc/codes"
)

type NamespaceServer struct {
	kubeConfig *v1.Config
}

func NewNamespaceServer(kubeConfig *v1.Config) *NamespaceServer {
	return &NamespaceServer{kubeConfig: kubeConfig}
}

func apiNamespace(ns *v1.Namespace) (namespace *api.Namespace) {
	namespace = &api.Namespace{
		Name: ns.Name,
	}

	return
}

func (s *NamespaceServer) ListNamespaces(ctx context.Context, empty *empty.Empty) (*api.ListNamespacesResponse, error) {
	client, err := v1.NewClient(s.kubeConfig, "")
	if err != nil {
		return nil, util.NewUserError(codes.PermissionDenied, "Permission denied.")
	}

	namespaces, err := client.ListNamespaces()
	if errors.As(err, &userError) {
		return nil, userError.GRPCError()
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

package server

import (
	"context"
	"math"
	"strings"

	api "github.com/onepanelio/core/api/gen"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/server/auth"
)

// NamespaceServer is an implementation of the grpc NamespaceServer
type NamespaceServer struct {
	api.UnimplementedNamespaceServiceServer
}

// NewNamespaceServer creates a new NamespaceServer
func NewNamespaceServer() *NamespaceServer {
	return &NamespaceServer{}
}

func apiNamespace(ns *v1.Namespace) (namespace *api.Namespace) {
	namespace = &api.Namespace{
		Name: ns.Name,
	}

	return
}

// ListNamespaces returns a list of all namespaces available in the system
func (s *NamespaceServer) ListNamespaces(ctx context.Context, req *api.ListNamespacesRequest) (*api.ListNamespacesResponse, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, "", "list", "", "namespaces", "")
	if err != nil || !allowed {
		return nil, err
	}

	if req.PageSize <= 0 {
		req.PageSize = 15
	}

	namespaces, err := client.ListNamespaces()
	if err != nil {
		return nil, err
	}

	var apiNamespaces []*api.Namespace
	for _, ns := range namespaces {
		if req.Query == "" || (req.Query != "" && strings.Contains(ns.Name, req.Query)) {
			apiNamespaces = append(apiNamespaces, apiNamespace(ns))
		}
	}

	pages := int32(math.Ceil(float64(len(apiNamespaces)) / float64(req.PageSize)))
	if req.Page > pages {
		req.Page = pages
	}

	if req.Page <= 0 {
		req.Page = 1
	}

	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if end >= int32(len(apiNamespaces)) {
		end = int32(len(apiNamespaces))
	}

	return &api.ListNamespacesResponse{
		Count:      end - start,
		Namespaces: apiNamespaces[start:end],
		Page:       req.Page,
		Pages:      pages,
		TotalCount: int32(len(apiNamespaces)),
	}, nil
}

// CreateNamespace creates a new namespace in the system
func (s *NamespaceServer) CreateNamespace(ctx context.Context, createNamespace *api.CreateNamespaceRequest) (*api.Namespace, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, "", "create", "", "namespaces", "")
	if err != nil || !allowed {
		return nil, err
	}

	namespace, err := client.CreateNamespace(createNamespace.Namespace.SourceName, createNamespace.Namespace.Name)
	if err != nil {
		return nil, err
	}

	return &api.Namespace{
		Name:       namespace.Name,
		SourceName: createNamespace.Namespace.SourceName,
	}, nil
}

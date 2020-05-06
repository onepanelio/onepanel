package server

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/onepanelio/core/api"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/pkg/util/pagination"
	"github.com/onepanelio/core/pkg/util/ptr"
	"github.com/onepanelio/core/server/auth"
	"time"
)

type WorkspaceServer struct{}

func apiWorkspace(wt *v1.Workspace) *api.Workspace {
	res := &api.Workspace{
		Uid:       wt.UID,
		Name:      wt.Name,
		CreatedAt: wt.CreatedAt.UTC().Format(time.RFC3339),
	}

	if wt.WorkspaceTemplate != nil {
		res.WorkspaceTemplate = apiWorkspaceTemplate(wt.WorkspaceTemplate)
	}

	return res
}

func NewWorkspaceServer() *WorkspaceServer {
	return &WorkspaceServer{}
}

func (s *WorkspaceServer) CreateWorkspace(ctx context.Context, req *api.CreateWorkspaceRequest) (*api.Workspace, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "apps", "statefulsets", "")
	if err != nil || !allowed {
		return nil, err
	}

	workspace := &v1.Workspace{
		Name: req.Workspace.Name,
		WorkspaceTemplate: &v1.WorkspaceTemplate{
			UID:     req.Workspace.WorkspaceTemplate.Uid,
			Version: req.Workspace.WorkspaceTemplate.Version,
		},
	}
	for _, param := range req.Workspace.Parameters {
		workspace.Parameters = append(workspace.Parameters, v1.Parameter{
			Name:  param.Name,
			Value: ptr.String(param.Value),
		})
	}
	workspace, err = client.CreateWorkspace(req.Namespace, workspace)
	if err != nil {
		return nil, err
	}

	req.Workspace = apiWorkspace(workspace)

	return req.Workspace, nil
}

func (s *WorkspaceServer) UpdateWorkspaceStatus(ctx context.Context, req *api.UpdateWorkspaceStatusRequest) (*empty.Empty, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "update", "apps", "statefulsets", "")
	if err != nil || !allowed {
		return &empty.Empty{}, err
	}

	status := &v1.WorkspaceStatus{
		Phase: v1.WorkspacePhase(req.Status.Phase),
	}
	err = client.UpdateWorkspaceStatus(req.Namespace, req.Uid, status)

	return &empty.Empty{}, err
}

func (s *WorkspaceServer) ListWorkspaces(ctx context.Context, req *api.ListWorkspaceRequest) (*api.ListWorkspaceResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "list", "argoproj.io", "statefulsets", "")
	if err != nil || !allowed {
		return nil, err
	}

	paginator := pagination.NewRequest(req.Page, req.PageSize)
	workspaces, err := client.ListWorkspaces(req.Namespace, &paginator)
	if err != nil {
		return nil, err
	}

	var apiWorkspaces []*api.Workspace
	for _, w := range workspaces {
		apiWorkspaces = append(apiWorkspaces, apiWorkspace(w))
	}

	return &api.ListWorkspaceResponse{
		Count:      int32(len(apiWorkspaces)),
		Workspaces: apiWorkspaces,
	}, nil
}

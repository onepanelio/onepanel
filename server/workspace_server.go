package server

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/onepanelio/core/api"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/pkg/util/pagination"
	"github.com/onepanelio/core/pkg/util/ptr"
	"github.com/onepanelio/core/server/auth"
	"github.com/onepanelio/core/server/converter"
	"time"
)

type WorkspaceServer struct{}

func apiWorkspace(wt *v1.Workspace) *api.Workspace {
	res := &api.Workspace{
		Uid:       wt.UID,
		Name:      wt.Name,
		CreatedAt: wt.CreatedAt.UTC().Format(time.RFC3339),
	}
	if len(wt.Labels) > 0 {
		res.Labels = converter.MappingToKeyValue(wt.Labels)
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
		WorkspaceTemplate: &v1.WorkspaceTemplate{
			UID:     req.WorkspaceTemplateUid,
			Version: req.WorkspaceTemplateVersion,
		},
		Labels: converter.APIKeyValueToLabel(req.Labels),
	}
	for _, param := range req.Parameters {
		if param.Type == "input.hidden" {
			continue
		}

		if param.Name == "sys-name" {
			workspace.Name = param.Value
		}

		workspace.Parameters = append(workspace.Parameters, v1.Parameter{
			Name:  param.Name,
			Value: ptr.String(param.Value),
		})
	}
	workspace, err = client.CreateWorkspace(req.Namespace, workspace)
	if err != nil {
		return nil, err
	}

	apiWorkspace := apiWorkspace(workspace)

	return apiWorkspace, nil
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

func (s *WorkspaceServer) PauseWorkspace(ctx context.Context, req *api.PauseWorkspaceRequest) (*empty.Empty, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "update", "apps", "statefulsets", "")
	if err != nil || !allowed {
		return &empty.Empty{}, err
	}

	err = client.PauseWorkspace(req.Namespace, req.Uid)

	return &empty.Empty{}, err
}

func (s *WorkspaceServer) DeleteWorkspace(ctx context.Context, req *api.DeleteWorkspaceRequest) (*empty.Empty, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "delete", "apps", "statefulsets", "")
	if err != nil || !allowed {
		return &empty.Empty{}, err
	}

	err = client.DeleteWorkspace(req.Namespace, req.Uid)

	return &empty.Empty{}, err
}

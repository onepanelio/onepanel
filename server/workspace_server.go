package server

import (
	"context"
	"github.com/onepanelio/core/api"
	v1 "github.com/onepanelio/core/pkg"
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
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "apps/v1", "statefulsets", "")
	if err != nil || !allowed {
		return nil, err
	}

	workspace := &v1.Workspace{
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

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
	"strings"
	"time"
)

type WorkspaceServer struct{}

func apiWorkspace(wt *v1.Workspace, config map[string]string) *api.Workspace {
	protocol := "http://"
	onepanelApiUrl := config["ONEPANEL_API_URL"]
	if strings.HasPrefix(onepanelApiUrl, "https://") {
		protocol = "https://"
	}

	res := &api.Workspace{
		Uid:       wt.UID,
		Name:      wt.Name,
		CreatedAt: wt.CreatedAt.UTC().Format(time.RFC3339),
		Url:       protocol + wt.URL,
	}

	res.Status = &api.WorkspaceStatus{
		Phase: string(wt.Status.Phase),
	}

	if wt.Status.StartedAt != nil {
		res.Status.StartedAt = wt.Status.StartedAt.UTC().Format(time.RFC3339)
	}

	if wt.Status.PausedAt != nil {
		res.Status.PausedAt = wt.Status.PausedAt.UTC().Format(time.RFC3339)
	}

	if wt.Status.TerminatedAt != nil {
		res.Status.TerminatedAt = wt.Status.TerminatedAt.UTC().Format(time.RFC3339)
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
			UID:     req.Body.WorkspaceTemplateUid,
			Version: req.Body.WorkspaceTemplateVersion,
			Labels:  converter.APIKeyValueToLabel(req.Body.Labels),
		},
	}

	for _, param := range req.Body.Parameters {
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

	sysConfig, err := client.GetSystemConfig()
	if err != nil {
		return nil, err
	}

	apiWorkspace := apiWorkspace(workspace, sysConfig)

	return apiWorkspace, nil
}

func (s *WorkspaceServer) GetWorkspace(ctx context.Context, req *api.GetWorkspaceRequest) (*api.Workspace, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "apps", "statefulsets", "")
	if err != nil || !allowed {
		return nil, err
	}

	workspace, err := client.GetWorkspace(req.Namespace, req.Uid)
	if err != nil {
		return nil, err
	}

	sysConfig, err := client.GetSystemConfig()
	if err != nil {
		return nil, err
	}

	apiWorkspace := apiWorkspace(workspace, sysConfig)

	return apiWorkspace, nil
}

func (s *WorkspaceServer) UpdateWorkspaceStatus(ctx context.Context, req *api.UpdateWorkspaceStatusRequest) (*empty.Empty, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "update", "apps", "statefulsets", req.Uid)
	if err != nil || !allowed {
		return &empty.Empty{}, err
	}

	status := &v1.WorkspaceStatus{
		Phase: v1.WorkspacePhase(req.Status.Phase),
	}
	err = client.UpdateWorkspaceStatus(req.Namespace, req.Uid, status)

	return &empty.Empty{}, err
}

func (s *WorkspaceServer) UpdateWorkspace(ctx context.Context, req *api.UpdateWorkspaceRequest) (*empty.Empty, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "update", "apps", "statefulsets", req.Uid)
	if err != nil || !allowed {
		return &empty.Empty{}, err
	}

	var parameters []v1.Parameter
	for _, param := range req.Body.Parameters {
		if param.Type == "input.hidden" {
			continue
		}

		parameters = append(parameters, v1.Parameter{
			Name:  param.Name,
			Value: ptr.String(param.Value),
		})
	}
	err = client.UpdateWorkspace(req.Namespace, req.Uid, parameters)

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

	sysConfig, err := client.GetSystemConfig()
	if err != nil {
		return nil, err
	}
	var apiWorkspaces []*api.Workspace
	for _, w := range workspaces {
		apiWorkspaces = append(apiWorkspaces, apiWorkspace(w, sysConfig))
	}

	count, err := client.CountWorkspaces(req.Namespace)
	if err != nil {
		return nil, err
	}

	return &api.ListWorkspaceResponse{
		Count:      int32(len(apiWorkspaces)),
		Workspaces: apiWorkspaces,
		Page:       int32(paginator.Page),
		Pages:      paginator.CalculatePages(count),
		TotalCount: int32(count),
	}, nil
}

func (s *WorkspaceServer) PauseWorkspace(ctx context.Context, req *api.PauseWorkspaceRequest) (*empty.Empty, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "update", "apps", "statefulsets", req.Uid)
	if err != nil || !allowed {
		return &empty.Empty{}, err
	}

	err = client.PauseWorkspace(req.Namespace, req.Uid)

	return &empty.Empty{}, err
}

func (s *WorkspaceServer) ResumeWorkspace(ctx context.Context, req *api.ResumeWorkspaceRequest) (*empty.Empty, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "update", "apps", "statefulsets", req.Uid)
	if err != nil || !allowed {
		return &empty.Empty{}, err
	}

	err = client.ResumeWorkspace(req.Namespace, req.Uid)

	return &empty.Empty{}, err
}

func (s *WorkspaceServer) DeleteWorkspace(ctx context.Context, req *api.DeleteWorkspaceRequest) (*empty.Empty, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "delete", "apps", "statefulsets", req.Uid)
	if err != nil || !allowed {
		return &empty.Empty{}, err
	}

	err = client.DeleteWorkspace(req.Namespace, req.Uid)

	return &empty.Empty{}, err
}

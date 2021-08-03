package server

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	api "github.com/onepanelio/core/api/gen"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/pkg/util"
	"github.com/onepanelio/core/pkg/util/ptr"
	"github.com/onepanelio/core/pkg/util/request"
	"github.com/onepanelio/core/pkg/util/request/pagination"
	requestSort "github.com/onepanelio/core/pkg/util/request/sort"
	"github.com/onepanelio/core/server/auth"
	"github.com/onepanelio/core/server/converter"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"time"
)

var reservedWorkspaceNames = map[string]bool{
	"modeldb": true,
}

// WorkspaceServer is an implementation of the grpc WorkspaceServer
type WorkspaceServer struct {
	api.UnimplementedWorkspaceServiceServer
}

// NewWorkspaceServer creates a new WorkspaceServer
func NewWorkspaceServer() *WorkspaceServer {
	return &WorkspaceServer{}
}

func apiWorkspace(wt *v1.Workspace, config v1.SystemConfig) *api.Workspace {
	if wt == nil {
		return nil
	}

	protocol := config.APIProtocol()
	domain := config.Domain()

	if protocol == nil {
		log.WithFields(log.Fields{
			"Method": "apiWorkspace",
			"Error":  "protocol is nil",
		}).Error("apiWorkspace")

		return nil
	}

	if domain == nil {
		log.WithFields(log.Fields{
			"Method": "apiWorkspace",
			"Error":  "domain is nil",
		}).Error("apiWorkspace")

		return nil
	}

	services, err := wt.WorkspaceTemplate.GetServices()
	if err != nil {
		return nil
	}

	apiServices := make([]*api.WorkspaceComponent, 0)
	for _, service := range services {
		apiServices = append(apiServices, &api.WorkspaceComponent{
			Name: service.Name,
			Url:  wt.GetURL(*protocol, *domain) + service.Path,
		})
	}

	res := &api.Workspace{
		Uid:                 wt.UID,
		Name:                wt.Name,
		CreatedAt:           wt.CreatedAt.UTC().Format(time.RFC3339),
		Url:                 wt.GetURL(*protocol, *domain),
		WorkspaceComponents: apiServices,
	}
	res.Parameters = converter.ParametersToAPI(wt.Parameters)

	nodePoolMap, err := config.NodePoolOptionsMap()
	if err != nil {
		log.WithFields(log.Fields{
			"Method": "apiWorkspace",
			"Error":  "Unable to get Node Pool Options Map",
		}).Error(err.Error())
		return nil
	}

	for _, parameter := range res.Parameters {
		if parameter.Name == "sys-node-pool" {
			mapVal := nodePoolMap[parameter.Value]
			res.MachineType = &api.MachineType{
				Name:  mapVal.Name,
				Value: mapVal.Value,
			}
		}
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

// CreateWorkspace create a workspace
func (s *WorkspaceServer) CreateWorkspace(ctx context.Context, req *api.CreateWorkspaceRequest) (*api.Workspace, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "onepanel.io", "workspaces", "")
	if err != nil || !allowed {
		return nil, err
	}

	workspace := &v1.Workspace{
		WorkspaceTemplate: &v1.WorkspaceTemplate{
			UID:     req.Body.WorkspaceTemplateUid,
			Version: req.Body.WorkspaceTemplateVersion,
		},
		Labels:      converter.APIKeyValueToLabel(req.Body.Labels),
		CaptureNode: req.Body.CaptureNode,
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

	if _, isReserved := reservedWorkspaceNames[workspace.Name]; isReserved {
		return nil, util.NewUserError(codes.AlreadyExists, "That name is reserved, choose a different name for the workspace.")
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

// GetWorkspace returns Workspace information
func (s *WorkspaceServer) GetWorkspace(ctx context.Context, req *api.GetWorkspaceRequest) (*api.Workspace, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "onepanel.io", "workspaces", req.Uid)
	if err != nil || !allowed {
		return nil, err
	}

	workspace, err := client.GetWorkspace(req.Namespace, req.Uid)
	if err != nil {
		return nil, err
	}
	if workspace == nil {
		return nil, util.NewUserError(codes.NotFound, "Not found")
	}

	sysConfig, err := client.GetSystemConfig()
	if err != nil {
		return nil, err
	}

	apiWorkspace := apiWorkspace(workspace, sysConfig)

	// We add the template parameters because they have additional information on the options for certain parameters.
	// e.g. select types need to know the options so they can display them, and the selected option properly.
	templateParameters, err := v1.ParseParametersFromManifest([]byte(workspace.WorkflowTemplateVersion.Manifest))
	if err != nil {
		return nil, err
	}

	templateParameters, err = sysConfig.UpdateNodePoolOptions(templateParameters)
	if err != nil {
		return nil, err
	}

	apiWorkspace.TemplateParameters = converter.ParametersToAPI(templateParameters)

	return apiWorkspace, nil
}

// UpdateWorkspaceStatus updates a given workspaces status such as running, paused, etc.
func (s *WorkspaceServer) UpdateWorkspaceStatus(ctx context.Context, req *api.UpdateWorkspaceStatusRequest) (*empty.Empty, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "update", "onepanel.io", "workspaces", req.Uid)
	if err != nil || !allowed {
		return &empty.Empty{}, err
	}

	status := &v1.WorkspaceStatus{
		Phase: v1.WorkspacePhase(req.Status.Phase),
	}

	err = client.UpdateWorkspaceStatus(req.Namespace, req.Uid, status)

	return &empty.Empty{}, err
}

// UpdateWorkspace updates a workspace's status
func (s *WorkspaceServer) UpdateWorkspace(ctx context.Context, req *api.UpdateWorkspaceRequest) (*empty.Empty, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "update", "onepanel.io", "workspaces", req.Uid)
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

// ListWorkspaces lists the current workspaces for a given namespace
func (s *WorkspaceServer) ListWorkspaces(ctx context.Context, req *api.ListWorkspaceRequest) (*api.ListWorkspaceResponse, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "list", "onepanel.io", "workspaces", "")
	if err != nil || !allowed {
		return nil, err
	}

	labelFilter, err := v1.LabelsFromString(req.Labels)
	if err != nil {
		return nil, err
	}
	reqSort, err := requestSort.New(req.Order)
	if err != nil {
		return nil, err
	}

	resourceRequest := &request.Request{
		Pagination: pagination.New(req.Page, req.PageSize),
		Filter: v1.WorkspaceFilter{
			Labels: labelFilter,
			Phase:  req.Phase,
		},
		Sort: reqSort,
	}

	workspaces, err := client.ListWorkspaces(req.Namespace, resourceRequest)
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

	count, err := client.CountWorkspaces(req.Namespace, resourceRequest)
	if err != nil {
		return nil, err
	}

	totalCount, err := client.CountWorkspaces(req.Namespace, nil)
	if err != nil {
		return nil, err
	}

	paginator := resourceRequest.Pagination
	return &api.ListWorkspaceResponse{
		Count:               int32(len(apiWorkspaces)),
		Workspaces:          apiWorkspaces,
		Page:                int32(paginator.Page),
		Pages:               paginator.CalculatePages(count),
		TotalCount:          int32(count),
		TotalAvailableCount: int32(totalCount),
	}, nil
}

// PauseWorkspace requests to pause a given workspace
func (s *WorkspaceServer) PauseWorkspace(ctx context.Context, req *api.PauseWorkspaceRequest) (*empty.Empty, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "update", "onepanel.io", "workspaces", req.Uid)
	if err != nil || !allowed {
		return &empty.Empty{}, err
	}

	err = client.PauseWorkspace(req.Namespace, req.Uid)

	return &empty.Empty{}, err
}

// ResumeWorkspace attempts to resume a workspace
func (s *WorkspaceServer) ResumeWorkspace(ctx context.Context, req *api.ResumeWorkspaceRequest) (*empty.Empty, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "update", "onepanel.io", "workspaces", req.Uid)
	if err != nil || !allowed {
		return &empty.Empty{}, err
	}

	var parameters []v1.Parameter
	for _, param := range req.Body.Parameters {
		parameters = append(parameters, v1.Parameter{
			Name:  param.Name,
			Value: ptr.String(param.Value),
		})
	}
	err = client.ResumeWorkspace(req.Namespace, req.Uid, parameters)

	return &empty.Empty{}, err
}

// DeleteWorkspace requests to delete a workspace
func (s *WorkspaceServer) DeleteWorkspace(ctx context.Context, req *api.DeleteWorkspaceRequest) (*empty.Empty, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "delete", "onepanel.io", "workspaces", req.Uid)
	if err != nil || !allowed {
		return &empty.Empty{}, err
	}

	err = client.DeleteWorkspace(req.Namespace, req.Uid)

	return &empty.Empty{}, err
}

// RetryLastWorkspaceAction will attempt the last action on the workspace again.
func (s *WorkspaceServer) RetryLastWorkspaceAction(ctx context.Context, req *api.RetryActionWorkspaceRequest) (*empty.Empty, error) {
	client := getClient(ctx)

	workspace, err := client.GetWorkspace(req.Namespace, req.Uid)
	if err != nil {
		return nil, err
	}
	if workspace == nil {
		return nil, util.NewUserError(codes.NotFound, "workspace not found")
	}

	verb := ""
	switch workspace.Status.Phase {
	case v1.WorkspaceFailedToLaunch:
		verb = "create"
	case v1.WorkspaceFailedToPause:
		verb = "update"
	case v1.WorkspaceFailedToResume:
		verb = "update"
	case v1.WorkspaceFailedToTerminate:
		verb = "delete"
	case v1.WorkspaceFailedToUpdate:
		verb = "update"
	default:
		return nil, util.NewUserError(codes.InvalidArgument, "Workspace is not in a failed state")
	}

	allowed, err := auth.IsAuthorized(client, req.Namespace, verb, "onepanel.io", "workspaces", req.Uid)
	if err != nil || !allowed {
		return &empty.Empty{}, err
	}

	switch workspace.Status.Phase {
	case v1.WorkspaceFailedToLaunch:
		if _, err := client.StartWorkspace(req.Namespace, workspace); err != nil {
			return nil, err
		}
	case v1.WorkspaceFailedToPause:
		if err := client.PauseWorkspace(req.Namespace, workspace.UID); err != nil {
			return nil, err
		}
	case v1.WorkspaceFailedToResume:
		if err := client.ResumeWorkspace(req.Namespace, workspace.UID, workspace.Parameters); err != nil {
			return nil, err
		}
	case v1.WorkspaceFailedToTerminate:
		if err := client.DeleteWorkspace(req.Namespace, workspace.UID); err != nil {
			return nil, err
		}
	case v1.WorkspaceFailedToUpdate:
		if err := client.UpdateWorkspace(req.Namespace, workspace.UID, workspace.Parameters); err != nil {
			return nil, err
		}
	default:
		return nil, util.NewUserError(codes.InvalidArgument, "Workspace is not in a failed state")
	}

	return &empty.Empty{}, err
}

// GetWorkspaceStatisticsForNamespace returns statistics on workflow executions for a given namespace
func (s *WorkspaceServer) GetWorkspaceStatisticsForNamespace(ctx context.Context, req *api.GetWorkspaceStatisticsForNamespaceRequest) (*api.GetWorkspaceStatisticsForNamespaceResponse, error) {
	client := getClient(ctx)

	allowed, err := auth.IsAuthorized(client, req.Namespace, "list", "onepanel.io", "workspaces", "")
	if err != nil || !allowed {
		return nil, err
	}

	report, err := client.GetWorkspaceStatisticsForNamespace(req.Namespace)
	if err != nil {
		return nil, err
	}

	return &api.GetWorkspaceStatisticsForNamespaceResponse{
		Stats: converter.WorkspaceStatisticsReportToAPI(report),
	}, nil
}

// GetWorkspaceContainerLogs returns logs for a given container name in a Workspace
func (s *WorkspaceServer) GetWorkspaceContainerLogs(req *api.GetWorkspaceContainerLogsRequest, stream api.WorkspaceService_GetWorkspaceContainerLogsServer) error {
	client := getClient(stream.Context())
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "onepanel.io", "workspaces", req.Uid)
	if err != nil || !allowed {
		return err
	}

	sinceTime := time.Unix(req.SinceTime, 0)
	watcher, err := client.GetWorkspaceContainerLogs(req.Namespace, req.Uid, req.ContainerName, sinceTime)
	if err != nil {
		return err
	}

	le := make([]*v1.LogEntry, 0)
	for {
		le = <-watcher
		if le == nil {
			break
		}

		apiLogEntries := make([]*api.LogEntry, len(le))
		for i, item := range le {
			apiLogEntries[i] = &api.LogEntry{
				Content: item.Content,
			}

			if item.Timestamp.After(time.Time{}) {
				apiLogEntries[i].Timestamp = item.Timestamp.Format(time.RFC3339)
			}
		}

		if err := stream.Send(&api.LogStreamResponse{
			LogEntries: apiLogEntries,
		}); err != nil {
			return err
		}
	}

	return nil
}

// ListWorkspacesField returns a list of all the distinct values of a field from Workspaces
func (s *WorkspaceServer) ListWorkspacesField(ctx context.Context, req *api.ListWorkspacesFieldRequest) (*api.ListWorkspacesFieldResponse, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "list", "onepanel.io", "workspaces", "")
	if err != nil || !allowed {
		return nil, err
	}

	values, err := client.ListWorkspacesField(req.Namespace, req.FieldName)
	if err != nil {
		return nil, err
	}

	return &api.ListWorkspacesFieldResponse{
		Values: values,
	}, nil
}

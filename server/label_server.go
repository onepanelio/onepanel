package server

import (
	"context"
	api "github.com/onepanelio/core/api/gen"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/server/auth"
	"github.com/onepanelio/core/server/converter"
)

func getGroupAndResourceByIdentifier(identifier string) (group, resource string) {
	group = "argoproj.io"
	switch identifier {
	case v1.TypeWorkflowTemplate:
		return group, "workflowtemplates"
	case v1.TypeWorkflowTemplateVersion:
		return group, "workflowtemplates"
	case v1.TypeWorkflowExecution:
		return group, "workflows"
	case v1.TypeCronWorkflow:
		return group, "cronworkflows"
	case v1.TypeWorkspace:
		return "onepanel.io", "workspaces"
	}

	return "", ""
}

func mapLabelsToKeyValue(labels []*v1.Label) []*api.KeyValue {
	result := make([]*api.KeyValue, len(labels))

	for i := range labels {
		result[i] = &api.KeyValue{
			Key:   labels[i].Key,
			Value: labels[i].Value,
		}
	}

	return result
}

func mapKeyValuesToMap(keyValues []*api.KeyValue) map[string]string {
	result := make(map[string]string)

	for _, keyValue := range keyValues {
		result[keyValue.Key] = keyValue.Value
	}

	return result
}

type LabelServer struct {
	api.UnimplementedLabelServiceServer
}

func NewLabelServer() *LabelServer {
	return &LabelServer{}
}

func (s *LabelServer) GetLabels(ctx context.Context, req *api.GetLabelsRequest) (*api.GetLabelsResponse, error) {
	group, resource := getGroupAndResourceByIdentifier(req.Resource)

	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", group, resource, "")
	if err != nil || !allowed {
		return nil, err
	}

	labels, err := client.ListLabels(req.Resource, req.Uid)
	if err != nil {
		return nil, err
	}

	return &api.GetLabelsResponse{
		Labels: mapLabelsToKeyValue(labels),
	}, nil
}

func (s *LabelServer) AddLabels(ctx context.Context, req *api.AddLabelsRequest) (*api.GetLabelsResponse, error) {
	group, resource := getGroupAndResourceByIdentifier(req.Resource)

	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", group, resource, "")
	if err != nil || !allowed {
		return nil, err
	}

	labelsMap := mapKeyValuesToMap(req.Labels.Items)
	if err := client.AddLabels(req.Namespace, req.Resource, req.Uid, labelsMap); err != nil {
		return nil, err
	}

	labels, err := client.ListLabels(req.Resource, req.Uid)
	if err != nil {
		return nil, err
	}

	return &api.GetLabelsResponse{
		Labels: mapLabelsToKeyValue(labels),
	}, nil
}

func (s *LabelServer) ReplaceLabels(ctx context.Context, req *api.ReplaceLabelsRequest) (*api.GetLabelsResponse, error) {
	group, resource := getGroupAndResourceByIdentifier(req.Resource)

	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "update", group, resource, "")
	if err != nil || !allowed {
		return nil, err
	}

	labelsMap := mapKeyValuesToMap(req.Labels.Items)
	if err := client.ReplaceLabels(req.Namespace, req.Resource, req.Uid, labelsMap); err != nil {
		return nil, err
	}

	labels, err := client.ListLabels(req.Resource, req.Uid)
	if err != nil {
		return nil, err
	}

	return &api.GetLabelsResponse{
		Labels: mapLabelsToKeyValue(labels),
	}, nil
}

func (s *LabelServer) DeleteLabel(ctx context.Context, req *api.DeleteLabelRequest) (*api.GetLabelsResponse, error) {
	group, resource := getGroupAndResourceByIdentifier(req.Resource)

	client := getClient(ctx)
	// update verb here since we are not deleting the resource, but labels
	allowed, err := auth.IsAuthorized(client, req.Namespace, "update", group, resource, "")
	if err != nil || !allowed {
		return nil, err
	}

	labelsMap := make(map[string]string)
	labelsMap[req.Key] = "placeholder"

	if err := client.DeleteLabels(req.Namespace, req.Resource, req.Uid, labelsMap); err != nil {
		return nil, err
	}

	labels, err := client.ListLabels(req.Resource, req.Uid)
	if err != nil {
		return nil, err
	}

	return &api.GetLabelsResponse{
		Labels: mapLabelsToKeyValue(labels),
	}, nil
}

// GetAvailableLabels returns the labels available for a resource specified by the GetAvailableLabelsRequest
func (s *LabelServer) GetAvailableLabels(ctx context.Context, req *api.GetAvailableLabelsRequest) (*api.GetLabelsResponse, error) {
	group, resource := getGroupAndResourceByIdentifier(req.Resource)

	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", group, resource, "")
	if err != nil || !allowed {
		return nil, err
	}

	labels, err := client.ListAvailableLabels(&v1.SelectLabelsQuery{
		Table:     v1.TypeToTableName(req.Resource),
		Alias:     "l",
		Namespace: req.Namespace,
		KeyLike:   req.KeyLike + "%",
		Skip:      v1.SkipKeysFromString(req.SkipKeys),
	})
	if err != nil {
		return nil, err
	}

	return &api.GetLabelsResponse{
		Labels: converter.LabelsToKeyValues(labels),
	}, nil
}

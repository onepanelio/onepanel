package server

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	api "github.com/onepanelio/core/api/gen"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/server/auth"
	"time"
)

// ModelServer is an implementation of the grpc ModelServer
type ModelServer struct {
	api.UnimplementedModelServiceServer
}

// NewModelServer creates a new ModelServer
func NewModelServer() *ModelServer {
	return &ModelServer{}
}

// DeployModel deploys a model server with a model(s)
func (s *ModelServer) DeployModel(ctx context.Context, req *api.DeployModelRequest) (*empty.Empty, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "serving.kubeflow.org", "inferenceservices", "")
	if err != nil || !allowed {
		return nil, err
	}

	model := &v1.ModelDeployment{
		Name:      req.Name,
		Namespace: req.Namespace,
		Predictor: &v1.Predictor{
			Server: v1.PredictorServer{
				Name:       req.Predictor.Server.Name,
				StorageURI: req.Predictor.Server.StorageUri,
			},
		},
	}

	if req.Predictor.Server.RuntimeVersion != "" {
		model.Predictor.Server.RuntimeVersion = &req.Predictor.Server.RuntimeVersion
	}

	if req.Predictor.Server.Limits != nil {
		model.Predictor.Server.ResourceLimits = &v1.ResourceLimits{
			CPU:    req.Predictor.Server.Limits.Cpu,
			Memory: req.Predictor.Server.Limits.Memory,
		}
	}

	if req.Predictor.NodeSelector != nil {
		model.Predictor.NodeSelector = &v1.NodeSelector{
			Key:   req.Predictor.NodeSelector.Key,
			Value: req.Predictor.NodeSelector.Value,
		}
	}

	if req.Transformer != nil {
		model.Transformer = &v1.Transformer{}

		for _, container := range req.Transformer.Containers {
			modelContainer := v1.TransformerContainer{
				Image: container.Image,
				Name:  container.Name,
			}

			for _, env := range container.Env {
				modelContainer.Env = append(modelContainer.Env, v1.Env{
					Name:  env.Name,
					Value: env.Value,
				})
			}

			model.Transformer.Containers = append(model.Transformer.Containers, modelContainer)
		}
	}

	err = client.DeployModel(model)
	if err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}

// GetModelStatus returns the status of a model
func (s *ModelServer) GetModelStatus(ctx context.Context, req *api.ModelIdentifier) (*api.ModelStatus, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "serving.kubeflow.org", "inferenceservices", req.Name)
	if err != nil || !allowed {
		return nil, err
	}

	status, err := client.GetModelStatus(req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}

	apiConditions := make([]*api.ModelCondition, len(status.Conditions))
	for i := range status.Conditions {
		condition := status.Conditions[i]
		apiConditions[i] = &api.ModelCondition{
			LastTransitionTime: condition.LastTransitionTime.Format(time.RFC3339),
			Status:             condition.Status,
			Type:               condition.Type,
		}
	}

	return &api.ModelStatus{
		Ready:      status.Ready,
		Conditions: apiConditions,
	}, nil
}

// DeleteModel deletes a model
func (s *ModelServer) DeleteModel(ctx context.Context, req *api.ModelIdentifier) (*empty.Empty, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "delete", "serving.kubeflow.org", "inferenceservices", req.Name)
	if err != nil || !allowed {
		return nil, err
	}

	err = client.DeleteModel(req.Namespace, req.Name)

	return &empty.Empty{}, err
}

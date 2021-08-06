package server

import (
	"context"
	api "github.com/onepanelio/core/api/gen"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/server/auth"
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
func (s *ModelServer) DeployModel(ctx context.Context, req *api.DeployModelRequest) (*api.DeployModelResponse, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "serving.kubeflow.org", "InferenceService", "")
	if err != nil || !allowed {
		return nil, err
	}

	model := &v1.ModelDeployment{
		Name:      req.Name,
		Namespace: req.Namespace,
		PredictorServer: v1.PredictorServer{
			Name:           req.Predictor.Server.Name,
			RuntimeVersion: req.Predictor.Server.RuntimeVersion,
			StorageURI:     req.Predictor.Server.StorageUri,
			ResourceLimits: v1.ResourceLimits{
				CPU:    req.Predictor.Server.Limits.Cpu,
				Memory: req.Predictor.Server.Limits.Memory,
			},
			NodeSelector: v1.NodeSelector{
				Key:   req.Predictor.NodeSelector.Key,
				Value: req.Predictor.NodeSelector.Value,
			},
		},
	}

	err = client.DeployModel(model)
	if err != nil {
		return nil, err
	}

	return &api.DeployModelResponse{
		Name: "test",
	}, nil
}

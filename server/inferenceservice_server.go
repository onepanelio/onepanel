package server

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	api "github.com/onepanelio/core/api/gen"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/pkg/util"
	"github.com/onepanelio/core/server/auth"
	"google.golang.org/grpc/codes"
	"time"
)

// InferenceServiceServer is an implementation of the grpc InferenceServiceServer
type InferenceServiceServer struct {
	api.UnimplementedInferenceServiceServer
}

// NewInferenceService creates a new InferenceServiceServer
func NewInferenceService() *InferenceServiceServer {
	return &InferenceServiceServer{}
}

// CreateInferenceService deploys an inference service
func (s *InferenceServiceServer) CreateInferenceService(ctx context.Context, req *api.CreateInferenceServiceRequest) (*api.GetInferenceServiceResponse, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "serving.kubeflow.org", "inferenceservices", "")
	if err != nil || !allowed {
		return nil, err
	}

	if req.Predictor.Name == "" {
		return nil, util.NewUserError(codes.InvalidArgument, "missing key 'predictor.name'")
	}

	if req.Predictor.StorageUri == "" {
		return nil, util.NewUserError(codes.InvalidArgument, "missing key 'predictor.storageUri'")
	}

	if req.DefaultTransformerImage != "" && req.Transformer != nil {
		return nil, util.NewUserError(codes.InvalidArgument, "must set either transformerImage or transformer, but not both")
	}

	model := &v1.InferenceService{
		Name:      req.Name,
		Namespace: req.Namespace,
		Predictor: &v1.Predictor{
			Name:       req.Predictor.Name,
			StorageURI: req.Predictor.StorageUri,
		},
	}

	model.Predictor.RuntimeVersion = req.Predictor.RuntimeVersion
	model.Predictor.SetResources(req.Predictor.MinCpu, req.Predictor.MaxCpu, req.Predictor.MinMemory, req.Predictor.MaxMemory)
	if req.Predictor.NodeSelector != "" {
		sysConfig, err := client.GetSystemConfig()
		if err != nil {
			return nil, err
		}
		nodePoolLabel := sysConfig.NodePoolLabel()
		if nodePoolLabel == nil {
			return nil, fmt.Errorf("applicationNodePoolLabel not set")
		}
		model.Predictor.SetNodeSelector(*nodePoolLabel, req.Predictor.NodeSelector)
	}

	if req.Transformer != nil {
		model.Transformer = &v1.Transformer{}

		for i, container := range req.Transformer.Containers {
			modelContainer := v1.TransformerContainer{
				Image: container.Image,
			}

			if container.Name == "" {
				modelContainer.Name = fmt.Sprintf("kfserving-container-%v", i)
			} else {
				modelContainer.Name = container.Name
			}

			modelContainer.Resources = &v1.Resources{
				Requests: &v1.MachineResources{
					CPU:    req.Transformer.MinCpu,
					Memory: req.Transformer.MinMemory,
				},
				Limits: &v1.MachineResources{
					CPU:    req.Transformer.MaxCpu,
					Memory: req.Transformer.MaxMemory,
				},
			}

			for _, env := range container.Env {
				modelContainer.Env = append(modelContainer.Env, v1.Env{
					Name:  env.Name,
					Value: env.Value,
				})
			}

			if len(container.Env) == 0 {
				modelContainer.Env = []v1.Env{
					{
						Name:  "STORAGE_URI",
						Value: req.Predictor.StorageUri,
					},
					{
						Name:  "model",
						Value: req.Name,
					},
				}
			}

			model.Transformer.Containers = append(model.Transformer.Containers, modelContainer)
		}
	} else if req.DefaultTransformerImage != "" {
		model.Transformer = &v1.Transformer{
			Containers: []v1.TransformerContainer{
				{
					Image: req.DefaultTransformerImage,
					Name:  "kfserving-container",
					Env: []v1.Env{
						{
							Name:  "STORAGE_URI",
							Value: req.Predictor.StorageUri,
						},
						{
							Name:  "model",
							Value: req.Name,
						},
					},
				},
			},
		}
	}

	err = client.CreateInferenceService(model)
	if err != nil {
		return nil, err
	}

	status, err := client.GetModelStatus(req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}

	apiConditions := make([]*api.InferenceServiceCondition, len(status.Conditions))
	for i := range status.Conditions {
		condition := status.Conditions[i]
		apiConditions[i] = &api.InferenceServiceCondition{
			LastTransitionTime: condition.LastTransitionTime.Format(time.RFC3339),
			Status:             condition.Status,
			Type:               condition.Type,
		}
	}

	return &api.GetInferenceServiceResponse{
		Ready:      status.Ready,
		Conditions: apiConditions,
		PredictUrl: status.PredictURL,
	}, nil
}

// GetInferenceService returns the status of an inferenceservice
func (s *InferenceServiceServer) GetInferenceService(ctx context.Context, req *api.InferenceServiceIdentifier) (*api.GetInferenceServiceResponse, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "serving.kubeflow.org", "inferenceservices", req.Name)
	if err != nil || !allowed {
		return nil, err
	}

	status, err := client.GetModelStatus(req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}

	apiConditions := make([]*api.InferenceServiceCondition, len(status.Conditions))
	for i := range status.Conditions {
		condition := status.Conditions[i]
		apiConditions[i] = &api.InferenceServiceCondition{
			LastTransitionTime: condition.LastTransitionTime.Format(time.RFC3339),
			Status:             condition.Status,
			Type:               condition.Type,
		}
	}

	return &api.GetInferenceServiceResponse{
		Ready:      status.Ready,
		Conditions: apiConditions,
		PredictUrl: status.PredictURL,
	}, nil
}

// DeleteInferenceService deletes an inference service
func (s *InferenceServiceServer) DeleteInferenceService(ctx context.Context, req *api.InferenceServiceIdentifier) (*empty.Empty, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "delete", "serving.kubeflow.org", "inferenceservices", req.Name)
	if err != nil || !allowed {
		return nil, err
	}

	err = client.DeleteModel(req.Namespace, req.Name)

	return &empty.Empty{}, err
}

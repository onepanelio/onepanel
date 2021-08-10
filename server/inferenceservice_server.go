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
func (s *InferenceServiceServer) CreateInferenceService(ctx context.Context, req *api.CreateInferenceServiceRequest) (*empty.Empty, error) {
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

	if req.TransformerImage != "" && req.Transformer != nil {
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

	if req.Predictor.RuntimeVersion != "" {
		model.Predictor.RuntimeVersion = &req.Predictor.RuntimeVersion
	}

	if req.Predictor.MinCpu != "" || req.Predictor.MinMemory != "" {
		model.Predictor.ResourceLimits = &v1.MachineResources{}
		if req.Predictor.MinCpu != "" {
			model.Predictor.ResourceLimits.CPU = req.Predictor.MinCpu
		}
		if req.Predictor.MinMemory != "" {
			model.Predictor.ResourceLimits.Memory = req.Predictor.MinMemory
		}
	}

	if req.Predictor.MaxCpu != "" || req.Predictor.MaxMemory != "" {
		model.Predictor.ResourceRequests = &v1.MachineResources{}
		if req.Predictor.MaxCpu != "" {
			model.Predictor.ResourceRequests.CPU = req.Predictor.MaxCpu
		}
		if req.Predictor.MaxMemory != "" {
			model.Predictor.ResourceRequests.Memory = req.Predictor.MaxMemory
		}
	}

	if req.Predictor.NodeSelector != "" {
		model.Predictor.NodeSelector = &req.Predictor.NodeSelector
	}

	if req.Transformer != nil {
		model.Transformer = &v1.Transformer{}

		if req.Transformer.MinCpu != "" || req.Transformer.MinMemory != "" {
			model.Transformer.ResourceRequests = &v1.MachineResources{}
			if req.Transformer.MinCpu != "" {
				model.Transformer.ResourceRequests.CPU = req.Transformer.MinCpu
			}
			if req.Transformer.MinMemory != "" {
				model.Transformer.ResourceRequests.Memory = req.Transformer.MinMemory
			}
		}

		if req.Transformer.MaxCpu != "" || req.Transformer.MaxMemory != "" {
			model.Transformer.ResourceLimits = &v1.MachineResources{}
			if req.Transformer.MaxCpu != "" {
				model.Transformer.ResourceLimits.CPU = req.Transformer.MaxCpu
			}
			if req.Transformer.MinMemory != "" {
				model.Transformer.ResourceLimits.Memory = req.Transformer.MaxMemory
			}
		}

		for i, container := range req.Transformer.Containers {
			modelContainer := v1.TransformerContainer{
				Image: container.Image,
			}

			if container.Name == "" {
				modelContainer.Name = fmt.Sprintf("kfserving-container-%v", i)
			} else {
				modelContainer.Name = container.Name
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
	} else if req.TransformerImage != "" {
		model.Transformer = &v1.Transformer{
			Containers: []v1.TransformerContainer{
				{
					Image: req.TransformerImage,
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

	return &empty.Empty{}, nil
}

// GetInferenceServiceStatus returns the status of an inferenceservice
func (s *InferenceServiceServer) GetInferenceServiceStatus(ctx context.Context, req *api.InferenceServiceIdentifier) (*api.InferenceServiceStatus, error) {
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

	return &api.InferenceServiceStatus{
		Ready:      status.Ready,
		Conditions: apiConditions,
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

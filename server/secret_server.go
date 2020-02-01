package server

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/onepanelio/core/api"
	"github.com/onepanelio/core/manager"
	"github.com/onepanelio/core/model"
	"github.com/onepanelio/core/util"
	"google.golang.org/grpc/codes"
)

type SecretServer struct {
	resourceManager *manager.ResourceManager
}

func NewSecretServer(resourceManager *manager.ResourceManager) *SecretServer {
	return &SecretServer{resourceManager: resourceManager}
}

func (s *SecretServer) CreateSecret(ctx context.Context, req *api.CreateSecretRequest) (*empty.Empty, error) {
	err := s.resourceManager.CreateSecret(req.Namespace, &model.Secret{
		Name: req.Secret.Name,
		Data: req.Secret.Data,
	})
	if err != nil {
		return nil, util.NewUserError(codes.Unknown, "Unknown error.")
	}

	return &empty.Empty{}, nil
}

func (s *SecretServer) SecretExists(ctx context.Context, req *api.SecretExistsRequest) (secretExists *api.SecretExistsResponse, err error) {
	var secretExistsBool bool
	secretExistsBool, err = s.resourceManager.SecretExists(req.Namespace, req.SecretName)
	if err != nil {
		return &api.SecretExistsResponse{
			Exists: false,
		}, util.NewUserError(codes.Unknown, "Unknown error.")
	}

	return secretExistsResponse(secretExistsBool), nil
}

func secretExistsResponse(secretExists bool) (secretExistsResFilled *api.SecretExistsResponse) {
	secretExistsResFilled = &api.SecretExistsResponse{
		Exists: secretExists,
	}
	return
}

func (s *SecretServer) GetSecret(ctx context.Context, req *api.GetSecretRequest) (*api.Secret, error) {
	secret, err := s.resourceManager.GetSecret(req.Namespace, req.Name)
	if err != nil {
		return nil, util.NewUserError(codes.Unknown, "Unknown error.")
	}

	apiSecret := &api.Secret{
		Name: secret.Name,
		Data: secret.Data,
	}

	return apiSecret, nil
}

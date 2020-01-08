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

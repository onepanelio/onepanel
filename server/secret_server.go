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

func apiSecret(s *model.Secret) *api.Secret {
	return &api.Secret{
		Name: s.Name,
		Data: s.Data,
	}
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
	secretExistsBool, err = s.resourceManager.SecretExists(req.Namespace, req.Name)
	if err != nil {
		return &api.SecretExistsResponse{
			Exists: false,
		}, util.NewUserError(codes.Unknown, "Unknown error.")
	}

	return secretExistsResponse(secretExistsBool), nil
}

func (s *SecretServer) GetSecret(ctx context.Context, req *api.GetSecretRequest) (*api.Secret, error) {
	secret, err := s.resourceManager.GetSecret(req.Namespace, req.Name)
	if err != nil {
		return nil, util.NewUserError(codes.Unknown, err.Error())
	}

	return apiSecret(secret), nil
}

func (s *SecretServer) ListSecrets(ctx context.Context, req *api.GetSecretsRequest) (secrets *api.Secrets, err error) {
	var modelSecrets []*model.Secret
	modelSecrets, err = s.resourceManager.ListSecrets(req.Namespace)
	if err != nil {
		return nil, util.NewUserError(codes.Unknown, err.Error())
	}
	var apiSecrets []*api.Secret
	for _, rawSecret := range modelSecrets {
		apiSecrets = append(apiSecrets, apiSecret(rawSecret))
	}
	secrets = &api.Secrets{
		Secrets: apiSecrets,
	}
	return
}

func (s *SecretServer) DeleteSecret(ctx context.Context, req *api.DeleteSecretRequest) (deleted *api.DeleteSecretResponse, err error) {
	var isDeleted bool
	isDeleted, err = s.resourceManager.DeleteSecret(req.Namespace, req.Name)
	if err != nil {
		return &api.DeleteSecretResponse{
			Deleted: false,
		}, util.NewUserError(codes.Unknown, err.Error())
	}
	return &api.DeleteSecretResponse{
		Deleted: isDeleted,
	}, nil
}

func (s *SecretServer) DeleteSecretKey(ctx context.Context, req *api.DeleteSecretKeyRequest) (deleted *api.DeleteSecretKeyResponse, err error) {
	var isDeleted bool
	isDeleted, err = s.resourceManager.DeleteSecretKey(req.Namespace, req.Name, req.Key)
	if err != nil {
		return &api.DeleteSecretKeyResponse{
			Deleted: false,
		}, util.NewUserError(codes.Unknown, err.Error())
	}
	return &api.DeleteSecretKeyResponse{
		Deleted: isDeleted,
	}, nil
}

func (s *SecretServer) AddSecretKeyValue(ctx context.Context, req *api.AddSecretValueRequest) (updated *api.AddSecretValueResponse, err error) {
	var isAdded bool
	isAdded, err = s.resourceManager.AddSecretKeyValue(req.Namespace, req.Name, req.AddSecretBody.Key, req.AddSecretBody.Value)
	if err != nil {
		return &api.AddSecretValueResponse{
			Inserted: false,
		}, util.NewUserError(codes.Unknown, err.Error())
	}
	return &api.AddSecretValueResponse{
		Inserted: isAdded,
	}, nil
}

func (s *SecretServer) UpdateSecretKeyValue(ctx context.Context, req *api.UpdateSecretKeyValueRequest) (updated *api.UpdateSecretKeyValueResponse, err error) {
	var isUpdated bool
	isUpdated, err = s.resourceManager.UpdateSecretKeyValue(req.Namespace, req.Name, req.Key, req.Value)
	if err != nil {
		return &api.UpdateSecretKeyValueResponse{
			Updated: false,
		}, util.NewUserError(codes.Unknown, err.Error())
	}
	return &api.UpdateSecretKeyValueResponse{
		Updated: isUpdated,
	}, nil
}

func secretExistsResponse(secretExists bool) (secretExistsResFilled *api.SecretExistsResponse) {
	secretExistsResFilled = &api.SecretExistsResponse{
		Exists: secretExists,
	}
	return
}

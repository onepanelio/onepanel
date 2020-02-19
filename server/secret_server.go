package server

import (
	"context"
	"errors"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/onepanelio/core/api"
	"github.com/onepanelio/core/manager"
	"github.com/onepanelio/core/model"
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
	if errors.As(err, &userError) {
		return nil, userError.GRPCError()
	}
	return &empty.Empty{}, nil
}

func (s *SecretServer) SecretExists(ctx context.Context, req *api.SecretExistsRequest) (secretExists *api.SecretExistsResponse, err error) {
	var secretExistsBool bool
	secretExistsBool, err = s.resourceManager.SecretExists(req.Namespace, req.Name)
	if errors.As(err, &userError) {
		return &api.SecretExistsResponse{
			Exists: false,
		}, userError.GRPCError()
	}
	return &api.SecretExistsResponse{
		Exists: secretExistsBool,
	}, nil
}

func (s *SecretServer) GetSecret(ctx context.Context, req *api.GetSecretRequest) (*api.Secret, error) {
	secret, err := s.resourceManager.GetSecret(req.Namespace, req.Name)
	if errors.As(err, &userError) {
		return nil, userError.GRPCError()
	}
	return apiSecret(secret), nil
}

func (s *SecretServer) ListSecrets(ctx context.Context, req *api.ListSecretsRequest) (*api.ListSecretsResponse, error) {
	secrets, err := s.resourceManager.ListSecrets(req.Namespace)
	if errors.As(err, &userError) {
		return nil, userError.GRPCError()
	}

	var apiSecrets []*api.Secret
	for _, secret := range secrets {
		apiSecrets = append(apiSecrets, apiSecret(secret))
	}

	return &api.ListSecretsResponse{
		Count:   int32(len(apiSecrets)),
		Secrets: apiSecrets,
	}, nil
}

func (s *SecretServer) DeleteSecret(ctx context.Context, req *api.DeleteSecretRequest) (deleted *api.DeleteSecretResponse, err error) {
	var isDeleted bool
	isDeleted, err = s.resourceManager.DeleteSecret(req.Namespace, req.Name)
	if errors.As(err, &userError) {
		return &api.DeleteSecretResponse{
			Deleted: false,
		}, userError.GRPCError()
	}
	return &api.DeleteSecretResponse{
		Deleted: isDeleted,
	}, nil
}

func (s *SecretServer) DeleteSecretKey(ctx context.Context, req *api.DeleteSecretKeyRequest) (deleted *api.DeleteSecretKeyResponse, err error) {
	var isDeleted bool
	secret := model.Secret{
		Name: req.SecretName,
		Data: map[string]string{
			req.Key:"",
		},
	}
	isDeleted, err = s.resourceManager.DeleteSecretKey(req.Namespace, &secret)
	if err != nil {
		if errors.As(err, &userError) {
			return &api.DeleteSecretKeyResponse{
				Deleted: false,
			}, userError.GRPCError()
		}
	}
	return &api.DeleteSecretKeyResponse{
		Deleted: isDeleted,
	}, nil
}

func (s *SecretServer) AddSecretKeyValue(ctx context.Context, req *api.AddSecretKeyValueRequest) (updated *api.AddSecretKeyValueResponse, err error) {
	var isAdded bool
	secret := &model.Secret{
		Name: req.Secret.Name,
		Data: req.Secret.Data,
	}
	isAdded, err = s.resourceManager.AddSecretKeyValue(req.Namespace, secret)
	if err != nil {
		if errors.As(err, &userError) {
			return &api.AddSecretKeyValueResponse{
				Inserted: false,
			}, userError.GRPCError()
		}
	}
	return &api.AddSecretKeyValueResponse{
		Inserted: isAdded,
	}, nil
}

func (s *SecretServer) UpdateSecretKeyValue(ctx context.Context, req *api.UpdateSecretKeyValueRequest) (updated *api.UpdateSecretKeyValueResponse, err error) {
	var isUpdated bool
	secret := model.Secret{
		Name: req.Secret.Name,
		Data: req.Secret.Data,
	}
	isUpdated, err = s.resourceManager.UpdateSecretKeyValue(req.Namespace, &secret)
	if errors.As(err, &userError) {
		return &api.UpdateSecretKeyValueResponse{
			Updated: false,
		}, userError.GRPCError()
	}
	return &api.UpdateSecretKeyValueResponse{
		Updated: isUpdated,
	}, nil
}

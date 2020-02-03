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

func (s *SecretServer) GetSecret(ctx context.Context, req *api.GetSecretRequest) (secretGet *api.Secret, err error) {
	var secret *apiv1.Secret
	secret, err = s.resourceManager.GetSecret(req.Namespace, req.SecretName)
	if err != nil {
		return nil, util.NewUserError(codes.Unknown, err.Error())
	}
	secretGet, err = getSecret(secret)
	if err != nil {
		return nil, util.NewUserError(codes.Unknown, err.Error())
	}
	return secretGet, nil
}

func (s *SecretServer) GetSecrets(ctx context.Context, req *api.GetSecretsRequest) (secrets *api.Secrets, err error) {
	var rawSecrets []apiv1.Secret
	rawSecrets, err = s.resourceManager.GetSecrets(req.Namespace)
	if err != nil {
		return nil, util.NewUserError(codes.Unknown, err.Error())
	}
	var apiSecret *api.Secret
	var apiSecrets []*api.Secret
	for _, rawSecret := range rawSecrets {
		apiSecret, err = getSecret(&rawSecret)
		if err != nil {
			return nil, err
		}
		apiSecrets = append(apiSecrets, apiSecret)
	}
	secrets = &api.Secrets{
		Secrets: apiSecrets,
	}
	return
}

func getSecret(secret *apiv1.Secret) (secretGetFilled *api.Secret, err error) {
	var secretData map[string]string
	secretData = make(map[string]string)
	for key, data := range secret.Data {
		secretData[key] = string(data)

	}
	secretGetFilled = &api.Secret{
		Name: secret.Name,
		Data: secretData,
	}
	return secretGetFilled, nil
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

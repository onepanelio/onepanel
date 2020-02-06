package server

import (
	"context"
	"errors"

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
	if len(req.Secret.Data) == 0 {
		return &api.DeleteSecretKeyResponse{
			Deleted: false,
		}, util.NewUserError(codes.InvalidArgument, errors.New("Data cannot be empty").Error())
	}
	//Currently, support for 1 key only
	singleKey := ""
	for key := range req.Secret.Data {
		singleKey = key
		break
	}
	isDeleted, err = s.resourceManager.DeleteSecretKey(req.Namespace, req.Secret.Name, singleKey)
	if err != nil {
		return &api.DeleteSecretKeyResponse{
			Deleted: false,
		}, util.NewUserError(codes.Unknown, err.Error())
	}
	return &api.DeleteSecretKeyResponse{
		Deleted: isDeleted,
	}, nil
}

func (s *SecretServer) AddSecretKeyValue(ctx context.Context, req *api.AddSecretKeyValueRequest) (updated *api.AddSecretKeyValueResponse, err error) {
	var isAdded bool
	if len(req.Secret.Data) == 0 {
		return &api.AddSecretKeyValueResponse{
			Inserted: false,
		}, util.NewUserError(codes.InvalidArgument, errors.New("Data cannot be empty").Error())
	}
	//Currently, support for 1 key only
	singleKey := ""
	singleVal := ""
	for key, value := range req.Secret.Data {
		singleKey = key
		singleVal = value
		break
	}
	isAdded, err = s.resourceManager.AddSecretKeyValue(req.Namespace, req.Secret.Name, singleKey, singleVal)
	if err != nil {
		return &api.AddSecretKeyValueResponse{
			Inserted: false,
		}, util.NewUserError(codes.Unknown, err.Error())
	}
	return &api.AddSecretKeyValueResponse{
		Inserted: isAdded,
	}, nil
}

func (s *SecretServer) UpdateSecretKeyValue(ctx context.Context, req *api.UpdateSecretKeyValueRequest) (updated *api.UpdateSecretKeyValueResponse, err error) {
	var isUpdated bool
	isUpdated, err = s.resourceManager.UpdateSecretKeyValue(req.Namespace, req.Secret.Name, req.Secret.Data)
	if errors.As(err, &userError) {
		return &api.UpdateSecretKeyValueResponse{
			Updated: false,
		}, userError.GRPCError()
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

package server

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	api "github.com/onepanelio/core/api/gen"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/server/auth"
)

type SecretServer struct {
	api.UnimplementedSecretServiceServer
}

func NewSecretServer() *SecretServer {
	return &SecretServer{}
}

func apiSecret(s *v1.Secret) *api.Secret {
	return &api.Secret{
		Name: s.Name,
		Data: s.Data,
	}
}

func (s *SecretServer) CreateSecret(ctx context.Context, req *api.CreateSecretRequest) (*empty.Empty, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "", "secrets", "")
	if err != nil || !allowed {
		return nil, err
	}

	err = client.CreateSecret(req.Namespace, &v1.Secret{
		Name: req.Secret.Name,
		Data: req.Secret.Data,
	})
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (s *SecretServer) SecretExists(ctx context.Context, req *api.SecretExistsRequest) (secretExists *api.SecretExistsResponse, err error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "", "secrets", req.Name)
	if err != nil || !allowed {
		return nil, err
	}

	secretExistsBool, err := client.SecretExists(req.Namespace, req.Name)
	if err != nil {
		return &api.SecretExistsResponse{
			Exists: false,
		}, err
	}
	return &api.SecretExistsResponse{
		Exists: secretExistsBool,
	}, nil
}

func (s *SecretServer) GetSecret(ctx context.Context, req *api.GetSecretRequest) (*api.Secret, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "", "secrets", req.Name)
	if err != nil || !allowed {
		return nil, err
	}

	secret, err := client.GetSecret(req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}
	return apiSecret(secret), nil
}

func (s *SecretServer) ListSecrets(ctx context.Context, req *api.ListSecretsRequest) (*api.ListSecretsResponse, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "list", "", "secrets", "")
	if err != nil || !allowed {
		return nil, err
	}

	secrets, err := client.ListSecrets(req.Namespace)
	if err != nil {
		return nil, err
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
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "delete", "", "secrets", req.Name)
	if err != nil || !allowed {
		return nil, err
	}

	isDeleted, err := client.DeleteSecret(req.Namespace, req.Name)
	if err != nil {
		return &api.DeleteSecretResponse{
			Deleted: false,
		}, err
	}
	return &api.DeleteSecretResponse{
		Deleted: isDeleted,
	}, nil
}

func (s *SecretServer) DeleteSecretKey(ctx context.Context, req *api.DeleteSecretKeyRequest) (deleted *api.DeleteSecretKeyResponse, err error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "delete", "", "secrets", req.SecretName)
	if err != nil || !allowed {
		return nil, err
	}

	secret := v1.Secret{
		Name: req.SecretName,
		Data: map[string]string{
			req.Key: "",
		},
	}
	isDeleted, err := client.DeleteSecretKey(req.Namespace, &secret)
	if err != nil {
		if err != nil {
			return &api.DeleteSecretKeyResponse{
				Deleted: false,
			}, err
		}
	}
	return &api.DeleteSecretKeyResponse{
		Deleted: isDeleted,
	}, nil
}

func (s *SecretServer) AddSecretKeyValue(ctx context.Context, req *api.AddSecretKeyValueRequest) (updated *api.AddSecretKeyValueResponse, err error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "delete", "", "secrets", req.Secret.Name)
	if err != nil || !allowed {
		return nil, err
	}

	secret := &v1.Secret{
		Name: req.Secret.Name,
		Data: req.Secret.Data,
	}
	isAdded, err := client.AddSecretKeyValue(req.Namespace, secret)
	if err != nil {
		if err != nil {
			return &api.AddSecretKeyValueResponse{
				Inserted: false,
			}, err
		}
	}
	return &api.AddSecretKeyValueResponse{
		Inserted: isAdded,
	}, nil
}

func (s *SecretServer) UpdateSecretKeyValue(ctx context.Context, req *api.UpdateSecretKeyValueRequest) (updated *api.UpdateSecretKeyValueResponse, err error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "update", "", "secrets", req.Secret.Name)
	if err != nil || !allowed {
		return nil, err
	}

	secret := v1.Secret{
		Name: req.Secret.Name,
		Data: req.Secret.Data,
	}
	isUpdated, err := client.UpdateSecretKeyValue(req.Namespace, &secret)
	if err != nil {
		return &api.UpdateSecretKeyValueResponse{
			Updated: false,
		}, err
	}
	return &api.UpdateSecretKeyValueResponse{
		Updated: isUpdated,
	}, nil
}

package manager

import (
	"encoding/base64"
	"encoding/json"
	goerrors "errors"
	"github.com/onepanelio/core/model"
	"github.com/onepanelio/core/util"
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/api/errors"
)

func (r *ResourceManager) CreateSecret(namespace string, secret *model.Secret) (err error) {
	if err = r.kubeClient.CreateSecret(namespace, secret); err != nil {
		return util.NewUserError(codes.Unknown, "Secret was not created.")
	}
	return
}

func (r *ResourceManager) SecretExists(namespace string, name string) (exists bool, err error) {
	var foundSecret *model.Secret
	var statusError *errors.StatusError
	secret := model.Secret{
		Name: name,
	}
	foundSecret, err = r.kubeClient.SecretExists(namespace, &secret)
	if err != nil {
		if goerrors.As(err, &statusError) {
			if statusError.ErrStatus.Reason == "NotFound" {
				return false, util.NewUserError(codes.NotFound, "Secret Not Found.")
			}
			return false, util.NewUserError(codes.Unknown, "Error when checking existence of secret.")
		}
		return false, util.NewUserError(codes.Unknown, "Error when checking existence of secret.")
	}
	if foundSecret.Name == "" {
		return false, nil
	}
	return true, nil
}

func (r *ResourceManager) GetSecret(namespace, name string) (secret *model.Secret, err error) {
	secretGet := model.Secret{
		Name: name,
	}
	secret, err = r.kubeClient.GetSecret(namespace, &secretGet)
	var statusError *errors.StatusError
	if err != nil {
		if goerrors.As(err, &statusError) {
			if statusError.ErrStatus.Reason == "NotFound" {
				return secret, util.NewUserError(codes.NotFound, "Secret Not Found.")
			}
			return secret, util.NewUserError(codes.Unknown, "Error when getting secret.")
		}
		return secret, util.NewUserError(codes.Unknown, "Error when getting secret.")
	}
	if secret == nil {
		return nil, util.NewUserError(codes.Unknown, "Error when getting secret.")
	}
	return
}

func (r *ResourceManager) ListSecrets(namespace string) (secrets []*model.Secret, err error) {
	secrets, err = r.kubeClient.ListSecrets(namespace)
	if err != nil {
		return nil, util.NewUserError(codes.NotFound, "No secrets were found.")
	}

	return
}

func (r *ResourceManager) DeleteSecret(namespace string, name string) (deleted bool, err error) {
	secret := &model.Secret{
		Name: name,
	}
	err = r.kubeClient.DeleteSecret(namespace, secret)
	if err != nil {
		return false, util.NewUserError(codes.Unknown, "Secret unable to be deleted.")
	}
	return true, nil
}

func (r *ResourceManager) DeleteSecretKey(namespace string, secret *model.Secret) (deleted bool, err error) {
	if len(secret.Data) == 0 {
		return false, util.NewUserError(codes.InvalidArgument, "Data cannot be empty")
	}
	//Currently, support for 1 key only
	key := ""
	for dataKey := range secret.Data {
		key = dataKey
		break
	}
	//Check if the secret has the key to delete
	secretFound, err := r.GetSecret(namespace, secret.Name)
	if err != nil {
		return false, util.NewUserError(codes.NotFound, "Secret not found.")
	}
	secretDataKeyExists := false

	for secretDataKey := range secretFound.Data {
		if secretDataKey == key {
			secretDataKeyExists = true
			break
		}
	}

	if secretDataKeyExists {
		//  patchStringPath specifies a patch operation for a secret key.
		type patchStringPath struct {
			Op   string `json:"op"`
			Path string `json:"path"`
		}
		payload := []patchStringPath{{
			Op:   "remove",
			Path: "/data/" + key,
		}}
		payloadBytes, _ := json.Marshal(payload)
		err = r.kubeClient.DeleteSecretKey(namespace, secret, payloadBytes)
		if err != nil {
			return false, util.NewUserError(codes.Unknown, "Unable to delete key from Secret.")
		}
		return true, nil

	}
	return false, util.NewUserError(codes.NotFound, "Key not found in Secret.")
}

func (r *ResourceManager) AddSecretKeyValue(namespace string, name string, key string, value string) (inserted bool, err error) {
	return r.kubeClient.AddSecretKeyValue(namespace, name, key, value)
}

func (r *ResourceManager) UpdateSecretKeyValue(namespace string, secret *model.Secret) (updated bool, err error) {
	if len(secret.Data) == 0 {
		return false, util.NewUserError(codes.InvalidArgument, "data cannot be empty.")
	}
	//Currently, support for 1 key only
	key := ""
	value := ""
	for dataKey, dataValue := range secret.Data {
		key = dataKey
		value = dataValue
		break
	}

	//Check if the secret has the key to update
	secretFound, err := r.GetSecret(namespace, secret.Name)
	if err != nil {
		return false, util.NewUserError(codes.NotFound, "Unable to find secret.")
	}
	secretDataKeyExists := false
	for secretDataKey := range secretFound.Data {
		if secretDataKey == key {
			secretDataKeyExists = true
			break
		}
	}
	if !secretDataKeyExists {
		errorMsg := "Key: " + key + " not found in secret."
		return false, util.NewUserError(codes.NotFound, errorMsg)
	}
	//  patchStringPath specifies a patch operation for a secret key.
	type patchStringPathWithValue struct {
		Op    string `json:"op"`
		Path  string `json:"path"`
		Value string `json:"value"`
	}
	valueEnc := base64.StdEncoding.EncodeToString([]byte(value))
	payload := []patchStringPathWithValue{{
		Op:    "replace",
		Path:  "/data/" + key,
		Value: valueEnc,
	}}
	payloadBytes, _ := json.Marshal(payload)
	err = r.kubeClient.UpdateSecretKeyValue(namespace, secret.Name, payloadBytes)
	if err != nil {
		return false, util.NewUserError(codes.Unknown, "Unable to update secret key value.")
	}
	return true, nil
}

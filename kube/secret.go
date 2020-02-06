package kube

import (
	"encoding/base64"
	"encoding/json"
	goerrors "errors"

	"github.com/onepanelio/core/model"
	corev1 "k8s.io/api/core/v1"
	errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (c *Client) CreateSecret(namespace string, secret *model.Secret) (err error) {
	_, err = c.CoreV1().Secrets(namespace).Create(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secret.Name,
		},
		StringData: secret.Data,
	})

	return
}

func (c *Client) SecretExists(namespace string, name string) (exists bool, err error) {
	var foundSecret *corev1.Secret
	var statusError *errors.StatusError
	foundSecret, err = c.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if goerrors.As(err, &statusError) {
			if statusError.ErrStatus.Reason == "NotFound" {
				return false, nil
			}
			return false, err
		}
		return false, err
	}
	if foundSecret != nil {
		return true, nil
	}
	return false, nil
}

func (c *Client) GetSecret(namespace string, name string) (secretRes *model.Secret, err error) {
	var foundSecret *corev1.Secret
	foundSecret, err = c.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if err, ok := err.(*errors.StatusError); ok {
			if err.ErrStatus.Reason == "NotFound" {
				return nil, nil
			}
			return nil, err
		}
		return nil, err
	}
	if foundSecret != nil {
		secretRes = &model.Secret{
			Name: foundSecret.Name,
			Data: convertSecretToMap(foundSecret),
		}
		return secretRes, nil
	}
	return nil, nil
}

func (c *Client) ListSecrets(namespace string) (secrets []*model.Secret, err error) {
	secretsList, err := c.CoreV1().Secrets(namespace).List(metav1.ListOptions{})
	if err != nil {
		return
	}

	for _, secret := range secretsList.Items {
		secretModel := model.Secret{
			Name: secret.Name,
			Data: convertSecretToMap(&secret),
		}
		secrets = append(secrets, &secretModel)
	}

	return
}

func (c *Client) DeleteSecret(namespace string, name string) (deleted bool, err error) {
	err = c.CoreV1().Secrets(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *Client) DeleteSecretKey(namespace string, name string, key string) (deleted bool, err error) {
	//Check if the secret has the key to delete
	secretFound, secretFindErr := c.GetSecret(namespace, name)
	if secretFindErr != nil {
		return false, secretFindErr
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
		_, errSecret := c.CoreV1().Secrets(namespace).Patch(name, types.JSONPatchType, payloadBytes)
		if errSecret != nil {
			return false, errSecret
		}
		return true, nil
	}
	return true, nil
}

func (c *Client) AddSecretKeyValue(namespace string, name string, key string, value string) (inserted bool, err error) {
	//Check if the secret has the key already
	secretFound, secretFindErr := c.GetSecret(namespace, name)
	if secretFindErr != nil {
		return false, secretFindErr
	}

	if secretFound == nil {
		return false, goerrors.New("Secret was not found.")
	}

	if len(secretFound.Data) > 0 {
		secretDataKeyExists := false
		for secretDataKey := range secretFound.Data {
			if secretDataKey == key {
				secretDataKeyExists = true
				break
			}
		}
		if secretDataKeyExists {
			errorMsg := "Key: " + key + " already exists in secret."
			return false, goerrors.New(errorMsg)
		}
	}
	//  patchStringPathAddNode specifies an add operation for a node
	type patchStringPathAddNode struct {
		Op    string            `json:"op"`
		Path  string            `json:"path"`
		Value map[string]string `json:"value"`
	}

	// "/data" may not exist due to 0 items. Create it so we can add an element.
	if len(secretFound.Data) == 0 {
		valMap := make(map[string]string)
		valueEnc := base64.StdEncoding.EncodeToString([]byte(value))
		valMap[key] = valueEnc
		payloadAddNode := []patchStringPathAddNode{{
			Op:    "add",
			Path:  "/data",
			Value: valMap,
		}}
		payloadAddNodeBytes, err := json.Marshal(payloadAddNode)
		if err != nil {
			return false, err
		}
		_, errSecret := c.CoreV1().Secrets(namespace).Patch(name, types.JSONPatchType, payloadAddNodeBytes)
		if errSecret != nil {
			return false, errSecret
		}
		return true, nil
	}
	//  patchStringPathAddKeyValue specifies an add operation, a key and value
	type patchStringPathAddKeyValue struct {
		Op    string `json:"op"`
		Path  string `json:"path"`
		Value string `json:"value"`
	}
	valueEnc := base64.StdEncoding.EncodeToString([]byte(value))
	payload := []patchStringPathAddKeyValue{{
		Op:    "add",
		Path:  "/data/" + key,
		Value: valueEnc,
	}}
	payloadBytes, _ := json.Marshal(payload)
	_, errSecret := c.CoreV1().Secrets(namespace).Patch(name, types.JSONPatchType, payloadBytes)
	if errSecret != nil {
		return false, errSecret
	}
	return true, nil
}

func (c *Client) UpdateSecretKeyValue(namespace string, name string, payloadBytes []byte) (err error) {
	_, err = c.CoreV1().Secrets(namespace).Patch(name, types.JSONPatchType, payloadBytes)
	return
}

func convertSecretToMap(foundSecret *corev1.Secret) (modelData map[string]string) {
	modelData = make(map[string]string)
	for secretKey, secretData := range foundSecret.Data {
		modelData[secretKey] = string(secretData)
	}
	return modelData
}

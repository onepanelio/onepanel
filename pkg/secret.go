package v1

import (
	"encoding/base64"
	"encoding/json"
	goerrors "errors"

	"github.com/onepanelio/core/util"
	"github.com/onepanelio/core/util/logging"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (c *Client) CreateSecret(namespace string, secret *Secret) (err error) {
	_, err = c.CoreV1().Secrets(namespace).Create(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secret.Name,
		},
		Data: secret.Data,
	})
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Secret":    secret,
			"Error":     err.Error(),
		}).Error("Error creating secret.")
		return util.NewUserError(codes.Unknown, "Secret was not created.")
	}
	return
}

func (c *Client) SecretExists(namespace string, name string) (exists bool, err error) {
	foundSecret, err := c.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Error":     err.Error(),
		}).Error("Secret Exists error.")

		var statusError *errors.StatusError
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

func (c *Client) GetSecret(namespace, name string) (secret *Secret, err error) {
	s, err := c.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Error":     err.Error(),
		}).Error("Secret not found error.")

		var statusError *errors.StatusError
		if goerrors.As(err, &statusError) {
			if statusError.ErrStatus.Reason == "NotFound" {
				return nil, util.NewUserError(codes.NotFound, "Secret Not Found.")
			}
			return nil, util.NewUserError(codes.Unknown, "Error when getting secret.")
		}
		return nil, util.NewUserError(codes.Unknown, "Error when getting secret.")
	}
	if s == nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Error":     "Secret is nil.",
		}).Error("Error getting secret.")
		return nil, util.NewUserError(codes.Unknown, "Error when getting secret.")
	}

	secret = &Secret{
		Name: s.Name,
		Data: s.Data,
	}
	return
}

func (c *Client) ListSecrets(namespace string) (secrets []*Secret, err error) {
	secretsList, err := c.CoreV1().Secrets(namespace).List(metav1.ListOptions{})
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Error":     err.Error(),
		}).Error("No secrets were found.")
		return nil, util.NewUserError(codes.NotFound, "No secrets were found.")
	}

	for _, s := range secretsList.Items {
		secret := Secret{
			Name: s.Name,
			Data: s.Data,
		}
		secrets = append(secrets, &secret)
	}

	return
}

func (c *Client) DeleteSecret(namespace string, name string) (deleted bool, err error) {
	err = c.CoreV1().Secrets(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Error":     err.Error(),
		}).Error("Unable to delete a secret.")
		return false, util.NewUserError(codes.Unknown, "Secret unable to be deleted.")
	}
	return true, nil
}

func (c *Client) DeleteSecretKey(namespace string, secret *Secret) (deleted bool, err error) {
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
	secretFound, err := c.GetSecret(namespace, secret.Name)
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Secret":    secret,
			"Error":     err.Error(),
		}).Error("Error with getting a secret.")
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
		_, err = c.CoreV1().Secrets(namespace).Patch(secret.Name, types.JSONPatchType, payloadBytes)
		if err != nil {
			logging.Logger.Log.WithFields(log.Fields{
				"Namespace": namespace,
				"Secret":    secret,
				"Error":     err.Error(),
			}).Error("Unable to a key from a secret.")
			return false, util.NewUserError(codes.Unknown, "Unable to delete key from Secret.")
		}
		return true, nil

	}
	return false, util.NewUserError(codes.NotFound, "Key not found in Secret.")
}

func (c *Client) AddSecretKeyValue(namespace string, secret *Secret) (inserted bool, err error) {
	if len(secret.Data) == 0 {
		return false, util.NewUserError(codes.InvalidArgument, "Data cannot be empty")
	}
	//Currently, support for 1 key only

	var (
		key   string
		value []byte
	)
	for dataKey, dataValue := range secret.Data {
		key = dataKey
		value = dataValue
		break
	}

	secretFound, err := c.GetSecret(namespace, secret.Name)
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Secret":    secret,
			"Error":     err.Error(),
		}).Error("Unable to find the secret.")
		return false, util.NewUserError(codes.NotFound, "Secret not found.")
	}

	if secretFound == nil {
		return false, util.NewUserError(codes.NotFound, "Secret not found.")
	}
	//Check if the secret has the key already
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
			return false, util.NewUserError(codes.AlreadyExists, errorMsg)
		}
	}

	//  patchStringPathAddNode specifies an add operation for a node
	type patchStringPathAddNode struct {
		Op    string            `json:"op"`
		Path  string            `json:"path"`
		Value map[string]string `json:"value"`
	}
	var payload []byte
	// "/data" may not exist due to 0 items. Create it with our new key and value
	if len(secretFound.Data) == 0 {
		valMap := make(map[string]string)
		valueEnc := base64.StdEncoding.EncodeToString([]byte(value))
		valMap[key] = valueEnc
		payloadAddNode := []patchStringPathAddNode{{
			Op:    "add",
			Path:  "/data",
			Value: valMap,
		}}
		payload, err = json.Marshal(payloadAddNode)
		if err != nil {
			logging.Logger.Log.WithFields(log.Fields{
				"Namespace": namespace,
				"Secret":    secret,
				"Error":     err.Error(),
			}).Error("Error building JSON.")
			return false, util.NewUserError(codes.InvalidArgument, "Error building JSON.")
		}
	} else {
		//  patchStringPathAddKeyValue specifies an add operation, a key and value
		type patchStringPathAddKeyValue struct {
			Op    string `json:"op"`
			Path  string `json:"path"`
			Value string `json:"value"`
		}
		valueEnc := base64.StdEncoding.EncodeToString([]byte(value))
		payloadAddData := []patchStringPathAddKeyValue{{
			Op:    "add",
			Path:  "/data/" + key,
			Value: valueEnc,
		}}
		payload, err = json.Marshal(payloadAddData)
		if err != nil {
			logging.Logger.Log.WithFields(log.Fields{
				"Namespace": namespace,
				"Secret":    secret,
				"Error":     err.Error(),
			}).Error("Error building JSON.")
			return false, util.NewUserError(codes.InvalidArgument, "Error building JSON.")
		}
	}
	_, err = c.CoreV1().Secrets(namespace).Patch(secret.Name, types.JSONPatchType, payload)
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Secret":    secret,
			"Error":     err.Error(),
		}).Error("Error adding key and value to Secret.")
		return false, util.NewUserError(codes.Unknown, "Error adding key and value to Secret.")
	}
	return true, nil
}

func (c *Client) UpdateSecretKeyValue(namespace string, secret *Secret) (updated bool, err error) {
	if len(secret.Data) == 0 {
		return false, util.NewUserError(codes.InvalidArgument, "data cannot be empty.")
	}
	//Currently, support for 1 key only
	var (
		key   string
		value []byte
	)
	for dataKey, dataValue := range secret.Data {
		key = dataKey
		value = dataValue
		break
	}

	//Check if the secret has the key to update
	secretFound, err := c.GetSecret(namespace, secret.Name)
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Secret":    secret,
			"Error":     err.Error(),
		}).Error("Unable to find secret.")
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
	_, err = c.CoreV1().Secrets(namespace).Patch(secret.Name, types.JSONPatchType, payloadBytes)
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Secret":    secret,
			"Error":     err.Error(),
		}).Error("Unable to update secret key value.")
		return false, util.NewUserError(codes.Unknown, "Unable to update secret key value.")
	}
	return true, nil
}

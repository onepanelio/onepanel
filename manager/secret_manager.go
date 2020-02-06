package manager

import (
	"github.com/onepanelio/core/model"
	corev1 "k8s.io/api/core/v1"
)

func (r *ResourceManager) CreateSecret(namespace string, secret *model.Secret) (err error) {
	return r.kubeClient.CreateSecret(namespace, secret)
}

func (r *ResourceManager) SecretExists(namespace string, name string) (exists bool, err error) {
	return r.kubeClient.SecretExists(namespace, name)
}

func (r *ResourceManager) GetSecret(namespace, name string) (secret *model.Secret, err error) {
	return r.kubeClient.GetSecret(namespace, name)
}

func (r *ResourceManager) ListSecrets(namespace string) (secrets []*model.Secret, err error) {
	secretList, err := r.kubeClient.ListSecrets(namespace)
	if err != nil {
		return nil, err
	}
	for _, secret := range secretList.Items {
		secretModel := model.Secret{
			Name: secret.Name,
			Data: convertSecretToMap(&secret),
		}
		secrets = append(secrets, &secretModel)
	}
	return
}

func (r *ResourceManager) DeleteSecret(namespace string, name string) (deleted bool, err error) {
	return r.kubeClient.DeleteSecret(namespace, name)
}

func (r *ResourceManager) DeleteSecretKey(namespace string, name string, key string) (deleted bool, err error) {
	return r.kubeClient.DeleteSecretKey(namespace, name, key)
}

func (r *ResourceManager) AddSecretKeyValue(namespace string, name string, key string, value string) (inserted bool, err error) {
	return r.kubeClient.AddSecretKeyValue(namespace, name, key, value)
}

func (r *ResourceManager) UpdateSecretKeyValue(namespace string, name string, key string, value string) (updated bool, err error) {
	return r.kubeClient.UpdateSecretKeyValue(namespace, name, key, value)
}
func convertSecretToMap(foundSecret *corev1.Secret) (modelData map[string]string) {
	modelData = make(map[string]string)
	for secretKey, secretData := range foundSecret.Data {
		modelData[secretKey] = string(secretData)
	}
	return modelData
}

package manager

import (
	"github.com/onepanelio/core/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *ResourceManager) CreateSecret(namespace string, secret *model.Secret) (err error) {
	return r.kubeClient.CreateSecret(namespace, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secret.Name,
		},
		StringData: secret.Data,
	})
}

func (r *ResourceManager) SecretExists(namespace string, name string) (exists bool, err error) {
	return r.kubeClient.SecretExists(namespace, name)
}

func (r *ResourceManager) GetSecret(namespace, name string) (secret *model.Secret, err error) {
	return r.kubeClient.GetSecret(namespace, name)
}

func (r *ResourceManager) ListSecrets(namespace string) (secrets []*model.Secret, err error) {
	return r.kubeClient.ListSecrets(namespace)
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

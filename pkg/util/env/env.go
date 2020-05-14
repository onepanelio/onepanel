package env

import (
	corev1 "k8s.io/api/core/v1"
	"os"
)

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func PrependEnvToContainer(container *corev1.Container, name, value string) {
	for _, e := range container.Env {
		if e.Name == name {
			return
		}
	}
	container.Env = append([]corev1.EnvVar{
		{
			Name:  name,
			Value: value,
		},
	}, container.Env...)
}

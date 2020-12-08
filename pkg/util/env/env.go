package env

import (
	"github.com/onepanelio/core/pkg/util/ptr"
	corev1 "k8s.io/api/core/v1"
	"os"
)

const (
	DefaultEnvironmentVariableSecret = "onepanel-default-env"
)

// GetEnv gets the environment variable value, or returns fallback if the environment variable does not exist
// Deprecated: use Get instead
func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// Get gets the environment variable value, or returns fallback if the environment variable does not exist
func Get(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func PrependEnvVarToContainer(container *corev1.Container, name, value string) {
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

func AddDefaultEnvVarsToContainer(container *corev1.Container) {
	container.EnvFrom = append(container.EnvFrom, corev1.EnvFromSource{
		SecretRef: &corev1.SecretEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: DefaultEnvironmentVariableSecret,
			},
			Optional: ptr.Bool(true),
		},
	})
}

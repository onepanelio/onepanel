package v1

import corev1 "k8s.io/api/core/v1"

type ArtifactRepositoryS3Config struct {
	KeyFormat       string
	Bucket          string
	Endpoint        string
	Insecure        bool
	Region          string
	AccessKeySecret corev1.SecretKeySelector
	SecretKeySecret corev1.SecretKeySelector
	AccessKey       string
	Secretkey       string
}

type ArtifactRepositoryConfig struct {
	S3 *ArtifactRepositoryS3Config
}

type NamespaceConfig struct {
	ArtifactRepository ArtifactRepositoryConfig
}

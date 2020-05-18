package v1

type ArtifactRepositoryS3Config struct {
	KeyFormat       string
	Bucket          string
	Endpoint        string
	Insecure        string
	Region          string
	AccessKeySecret struct {
		Name string
		Key  string
	}
	SecretKeySecret struct {
		Name string
		Key  string
	}
	AccessKey string
	Secretkey string
}

type ArtifactRepositoryConfig struct {
	S3 *ArtifactRepositoryS3Config
}

type NamespaceConfig struct {
	ArtifactRepository ArtifactRepositoryConfig
}

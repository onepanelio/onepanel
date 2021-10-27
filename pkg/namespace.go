package v1

import (
	"encoding/base64"
	"fmt"
	"github.com/onepanelio/core/pkg/util"
	"github.com/onepanelio/core/pkg/util/data"
	"google.golang.org/grpc/codes"
	"io/ioutil"
	vapps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	vnet "k8s.io/api/networking/v1"
	v1rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"os"
	"path/filepath"
	"strings"
)

var onepanelEnabledLabelKey = "onepanel.io/enabled"

func replaceVariables(filepath string, replacements map[string]string) (string, error) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return "", err
	}

	dataStr := string(data)
	for key, value := range replacements {
		dataStr = strings.ReplaceAll(dataStr, key, value)
	}

	return dataStr, nil
}

func (c *Client) ListOnepanelEnabledNamespaces() (namespaces []*Namespace, err error) {
	namespaceList, err := c.CoreV1().Namespaces().List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", onepanelEnabledLabelKey, "true"),
	})
	if err != nil {
		return
	}

	for _, ns := range namespaceList.Items {
		namespaces = append(namespaces, &Namespace{
			Name:   ns.Name,
			Labels: ns.Labels,
		})
	}

	return
}

// GetNamespace gets the namespace from the cluster if it exists
func (c *Client) GetNamespace(name string) (namespace *Namespace, err error) {
	ns, err := c.CoreV1().Namespaces().Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	namespace = &Namespace{
		Name:   ns.Name,
		Labels: ns.Labels,
	}

	return
}

// ListNamespaces lists all of the onepanel enabled namespaces
func (c *Client) ListNamespaces() (namespaces []*Namespace, err error) {
	namespaceList, err := c.CoreV1().Namespaces().List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", onepanelEnabledLabelKey, "true"),
	})
	if err != nil {
		return
	}

	for _, ns := range namespaceList.Items {
		namespaces = append(namespaces, &Namespace{
			Name:   ns.Name,
			Labels: ns.Labels,
		})
	}

	return
}

// CreateNamespace creates a namespace named {{ name }} assuming the {{ sourceNamespace }} created it
func (c *Client) CreateNamespace(sourceNamespace, name string) (namespace *Namespace, err error) {
	newNamespace := name
	domain := *c.systemConfig.Domain()
	artifactRepositorySource, err := c.GetArtifactRepositorySource(sourceNamespace)
	if err != nil {
		return nil, err
	}

	config, err := c.GetNamespaceConfig(sourceNamespace)
	if err != nil {
		return nil, err
	}
	if config.ArtifactRepository.S3 == nil {
		return nil, util.NewUserError(codes.Internal, "S3 compatible artifact repository not set")
	}

	accessKey := config.ArtifactRepository.S3.AccessKey
	secretKey := config.ArtifactRepository.S3.Secretkey

	if err := c.createK8sNamespace(name); err != nil {
		return nil, err
	}

	if err := c.createNetworkPolicy(newNamespace); err != nil {
		return nil, err
	}

	if err := c.createIstioVirtualService(newNamespace, domain); err != nil {
		return nil, err
	}

	if err := c.createRole(newNamespace); err != nil {
		return nil, err
	}

	if err := c.createDefaultSecret(newNamespace); err != nil {
		return nil, err
	}

	if err := c.createSecretOnepanelDefaultNamespace(newNamespace, accessKey, secretKey); err != nil {
		return nil, err
	}

	if err := c.createProviderDependentMinioDeployment(newNamespace, artifactRepositorySource); err != nil {
		return nil, err
	}

	if err := c.createProviderDependentMinioService(newNamespace, artifactRepositorySource); err != nil {
		return nil, err
	}

	if err := c.createNamespaceConfigMap(sourceNamespace, newNamespace); err != nil {
		return nil, err
	}

	if err := c.createNamespaceClusterRoleBinding(newNamespace); err != nil {
		return nil, err
	}

	if err := c.createNamespaceRoleBinding(newNamespace); err != nil {
		return nil, err
	}

	if err := c.createNamespaceServiceAccount(newNamespace); err != nil {
		return nil, err
	}

	if err := c.createNamespaceModelClusterRoleBinding(newNamespace); err != nil {
		return nil, err
	}

	if err := c.createNamespaceTemplates(newNamespace); err != nil {
		return nil, err
	}

	namespace = &Namespace{
		Name: name,
	}

	return
}

func (c *Client) createK8sNamespace(name string) error {
	createNamespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"istio-injection":              "enabled",
				onepanelEnabledLabelKey:        "true",
				"app.kubernetes.io/component":  "onepanel",
				"app.kubernetes.io/instance":   "onepanel-v0.5.0",
				"app.kubernetes.io/managed-by": "onepanel-cli",
				"app.kubernetes.io/name":       "onepanel",
				"app.kubernetes.io/part-of":    "onepanel",
				"app.kubernetes.io/version":    "v0.5.0",
			},
		},
	}

	_, err := c.CoreV1().Namespaces().Create(createNamespace)
	if err != nil && strings.Contains(err.Error(), "already exists") {
		return util.NewUserError(codes.AlreadyExists, "Namespace '"+name+"' already exists")
	}

	return err
}

func (c *Client) createNetworkPolicy(namespace string) error {
	replacements := map[string]string{
		"$(applicationDefaultNamespace)": namespace,
	}

	dataStr, err := replaceVariables(filepath.Join("manifest", "networkpolicy-onepanel-defaultnamespace.json"), replacements)
	if err != nil {
		return err
	}
	data := []byte(dataStr)

	networkPolicy := &vnet.NetworkPolicy{}
	if err := json.Unmarshal(data, networkPolicy); err != nil {
		return err
	}

	_, err = c.NetworkingV1().NetworkPolicies(namespace).Create(networkPolicy)

	return err
}

func (c *Client) createIstioVirtualService(namespace, domain string) error {
	replacements := map[string]string{
		"$(applicationDefaultNamespace)": namespace,
		"$(applicationDomain)":           domain,
	}

	dataStr, err := replaceVariables(filepath.Join("manifest", "service-minio-onepanel.json"), replacements)
	if err != nil {
		return err
	}
	data := []byte(dataStr)

	return c.CreateVirtualService(namespace, data)
}

func (c *Client) createRole(namespace string) error {
	replacements := map[string]string{
		"$(applicationDefaultNamespace)": namespace,
	}

	dataStr, err := replaceVariables(filepath.Join("manifest", "role-onepanel-defaultnamespace.json"), replacements)
	if err != nil {
		return err
	}
	data := []byte(dataStr)

	role := &v1rbac.Role{}
	if err := json.Unmarshal(data, role); err != nil {
		return err
	}

	_, err = c.RbacV1().Roles(namespace).Create(role)

	return err
}

func (c *Client) createDefaultSecret(namespace string) error {
	replacements := map[string]string{
		"$(applicationDefaultNamespace)": namespace,
	}

	dataStr, err := replaceVariables(filepath.Join("manifest", "secret-onepanel-default-env-defaultnamespace.json"), replacements)
	if err != nil {
		return err
	}
	data := []byte(dataStr)

	secret := &v1.Secret{}
	if err := json.Unmarshal(data, secret); err != nil {
		return err
	}

	_, err = c.CoreV1().Secrets(namespace).Create(secret)

	return err
}

func (c *Client) createSecretOnepanelDefaultNamespace(namespace, accessKey, secretKey string) error {
	replacements := map[string]string{
		"$(applicationDefaultNamespace)":   namespace,
		"$(artifactRepositoryS3AccessKey)": base64.StdEncoding.EncodeToString([]byte(accessKey)),
		"$(artifactRepositoryS3SecretKey)": base64.StdEncoding.EncodeToString([]byte(secretKey)),
	}

	dataStr, err := replaceVariables(filepath.Join("manifest", "secret-onepanel-defaultnamespace.json"), replacements)
	if err != nil {
		return err
	}
	data := []byte(dataStr)

	secret := &v1.Secret{}
	if err := json.Unmarshal(data, secret); err != nil {
		return err
	}

	_, err = c.CoreV1().Secrets(namespace).Create(secret)

	return err
}

func (c *Client) createProviderDependentMinioDeployment(namespace, artifactRepositoryProvider string) error {
	replacements := map[string]string{
		"$(applicationDefaultNamespace)": namespace,
	}

	//// AWS S3 doesn't require a specific artifactRepositoryProvider
	if artifactRepositoryProvider == "s3" {
		return nil
	}

	dataStr, err := replaceVariables(filepath.Join("manifest", artifactRepositoryProvider, "deployment.json"), replacements)
	if err != nil {
		return err
	}
	data := []byte(dataStr)

	deployment := &vapps.Deployment{}
	if err := json.Unmarshal(data, deployment); err != nil {
		return err
	}

	_, err = c.AppsV1().Deployments(namespace).Create(deployment)

	return err
}

func (c *Client) createProviderDependentMinioService(namespace, artifactRepositoryProvider string) error {
	replacements := map[string]string{
		"$(applicationDefaultNamespace)": namespace,
	}

	// AWS S3 doesn't require a specific artifactRepositoryProvider
	if artifactRepositoryProvider == "s3" {
		return nil
	}

	dataStr, err := replaceVariables(filepath.Join("manifest", artifactRepositoryProvider, "service.json"), replacements)
	if err != nil {
		return err
	}
	data := []byte(dataStr)

	service := &v1.Service{}
	if err := json.Unmarshal(data, service); err != nil {
		return err
	}

	_, err = c.CoreV1().Services(namespace).Create(service)

	return err
}

func (c *Client) createNamespaceClusterRoleBinding(namespace string) error {
	replacements := map[string]string{
		"$(applicationDefaultNamespace)": namespace,
	}

	dataStr, err := replaceVariables(filepath.Join("manifest", "clusterrolebinding-onepanel-namespaces-defaultnamespace.json"), replacements)
	if err != nil {
		return err
	}
	data := []byte(dataStr)

	resource := &v1rbac.ClusterRoleBinding{}
	if err := json.Unmarshal(data, resource); err != nil {
		return err
	}

	resource.Name += "-" + namespace

	_, err = c.RbacV1().ClusterRoleBindings().Create(resource)

	return err
}

func (c *Client) createNamespaceRoleBinding(namespace string) error {
	replacements := map[string]string{
		"$(applicationDefaultNamespace)": namespace,
	}

	dataStr, err := replaceVariables(filepath.Join("manifest", "rolebinding-onepanel-defaultnamespace.json"), replacements)
	if err != nil {
		return err
	}
	data := []byte(dataStr)

	resource := &v1rbac.RoleBinding{}
	if err := json.Unmarshal(data, resource); err != nil {
		return err
	}

	_, err = c.RbacV1().RoleBindings(namespace).Create(resource)

	return err
}

func (c *Client) createNamespaceServiceAccount(namespace string) error {
	replacements := map[string]string{
		"$(applicationDefaultNamespace)": namespace,
	}

	dataStr, err := replaceVariables(filepath.Join("manifest", "service-account.json"), replacements)
	if err != nil {
		return err
	}
	data := []byte(dataStr)

	resource := &v1.ServiceAccount{}
	if err := json.Unmarshal(data, resource); err != nil {
		return err
	}

	_, err = c.CoreV1().ServiceAccounts(namespace).Create(resource)

	return err
}

func (c *Client) createNamespaceConfigMap(sourceNamespace, namespace string) error {
	sourceConfigMap, err := c.CoreV1().ConfigMaps(sourceNamespace).Get("onepanel", metav1.GetOptions{})
	if err != nil {
		return err
	}

	data := sourceConfigMap.Data["artifactRepository"]
	sourceKey := "minio-gateway." + sourceNamespace + ".svc.cluster.local:9000"
	replaceKey := "minio-gateway." + namespace + ".svc.cluster.local:9000"
	data = strings.ReplaceAll(data, sourceKey, replaceKey)
	sourceConfigMap.Data["artifactRepository"] = data

	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "onepanel",
			Namespace: namespace,
		},
		Data: sourceConfigMap.Data,
	}

	configMap.Namespace = namespace

	_, err = c.CoreV1().ConfigMaps(namespace).Create(configMap)

	return err
}

func (c *Client) createNamespaceModelClusterRoleBinding(namespace string) error {
	replacements := map[string]string{
		"$(applicationDefaultNamespace)": namespace,
	}

	dataStr, err := replaceVariables(filepath.Join("manifest", "clusterrolebinding-models.json"), replacements)
	if err != nil {
		return err
	}
	data := []byte(dataStr)

	resource := &v1rbac.ClusterRoleBinding{}
	if err := json.Unmarshal(data, resource); err != nil {
		return err
	}

	_, err = c.RbacV1().ClusterRoleBindings().Create(resource)

	return err
}

func (c *Client) createNamespaceTemplates(namespace string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	workflowDir := filepath.Join(wd, "db", "yaml")

	filepaths := make([]string, 0)

	err = filepath.Walk(workflowDir,
		func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				filepaths = append(filepaths, path)
			}

			return nil
		},
	)
	if err != nil {
		return err
	}

	for _, filename := range filepaths {
		manifest, err := data.ManifestFileFromFile(filename)
		if err != nil {
			return err
		}

		if manifest.Metadata.Kind == "Workflow" {
			if manifest.Metadata.Action == "create" {
				if err := c.createWorkflowTemplateFromGenericManifest(namespace, manifest); err != nil {
					return err
				}
			} else {
				if err := c.updateWorkflowTemplateManifest(namespace, manifest); err != nil {
					return err
				}
			}
		} else if manifest.Metadata.Kind == "Workspace" {
			if manifest.Metadata.Action == "create" {
				if err := c.createWorkspaceTemplateFromGenericManifest(namespace, manifest); err != nil {
					return err
				}
			} else {
				if err := c.updateWorkspaceTemplateManifest(namespace, manifest); err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("unknown manifest type for file %v", filename)
		}
	}

	return nil
}

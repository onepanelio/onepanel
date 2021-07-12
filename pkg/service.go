package v1

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// generateServiceURL generates the url that the service is located at
func (c *Client) generateServiceURL(namespace, name string) (string, error) {
	protocol := c.systemConfig.APIProtocol()
	if protocol == nil {
		return "", fmt.Errorf("unable to get the api protocol from the system config")
	}

	domain := c.systemConfig.Domain()
	if domain == nil {
		return "", fmt.Errorf("unable to get a domain from the system config")
	}

	// https://name--namespace.domain
	return fmt.Sprintf("%v%v--%v.%v", *protocol, name, namespace, *domain), nil
}

// ListServices finds all of the services in the given namespace
func (c *Client) ListServices(namespace string) ([]*Service, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace is empty")
	}

	labelSelect := fmt.Sprintf("%v=%v", "service.onepanel.io/part-of", "onepanel")

	serviceList, err := c.CoreV1().Services(namespace).List(ListOptions{
		LabelSelector: labelSelect,
	})
	if err != nil {
		return nil, err
	}

	services := make([]*Service, 0)
	for _, serviceItem := range serviceList.Items {
		serviceName := serviceItem.Labels["service.onepanel.io/name"]
		serviceURL, err := c.generateServiceURL(namespace, serviceName)
		if err != nil {
			return nil, err
		}

		services = append(services, &Service{
			Name: serviceName,
			URL:  serviceURL,
		})
	}

	return services, nil
}

// GetService gets a specific service identified by namespace, name.
// If it is not found, nil, nil is returned
func (c *Client) GetService(namespace, name string) (*Service, error) {
	labelSelect := fmt.Sprintf("%v=%v,%v=%v", "service.onepanel.io/part-of", "onepanel", "service.onepanel.io/name", name)

	serviceList, err := c.CoreV1().Services(namespace).List(ListOptions{
		LabelSelector: labelSelect,
	})
	if err != nil {
		return nil, err
	}

	if len(serviceList.Items) == 0 {
		return nil, nil
	}
	if len(serviceList.Items) > 1 {
		return nil, fmt.Errorf("non-unique result found for GetService %v,%v", namespace, name)
	}

	serviceItem := serviceList.Items[0]
	serviceName := serviceItem.Labels["service.onepanel.io/name"]
	serviceURL, err := c.generateServiceURL(namespace, serviceName)
	if err != nil {
		return nil, err
	}

	service := &Service{
		Name: serviceName,
		URL:  serviceURL,
	}

	return service, nil
}

// HasService checks if the cluster has a service available
func (c *Client) HasService(name string) (bool, error) {
	if name != "kfserving-system" {
		return false, fmt.Errorf("unsupported service")
	}

	if _, err := c.GetNamespace(name); err != nil {
		return false, err
	}

	// Check if deployment is there for the web app
	_, err := c.CoreV1().Pods("kfserving-system").List(metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/component=kfserving-models-web-app",
	})
	if err != nil {
		return false, err
	}

	return true, nil
}

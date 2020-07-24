package v1

import (
	"fmt"
)

// generateComponentURL generates the url that the component is located at
func (c *Client) generateComponentURL(namespace, name string) (string, error) {
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

// ListComponents finds all of the components in the given namespace
func (c *Client) ListComponents(namespace string) ([]*Component, error) {
	labelSelect := fmt.Sprintf("%v=%v", "component.onepanel.io/part-of", "onepanel")

	serviceList, err := c.CoreV1().Services(namespace).List(ListOptions{
		LabelSelector: labelSelect,
	})
	if err != nil {
		return nil, err
	}

	components := make([]*Component, 0)
	for _, serviceItem := range serviceList.Items {
		componentName := serviceItem.Labels["component.onepanel.io/name"]
		componentURL, err := c.generateComponentURL(namespace, componentName)
		if err != nil {
			return nil, err
		}

		components = append(components, &Component{
			Name: componentName,
			URL:  componentURL,
		})
	}

	return components, nil
}

// GetComponent gets a specific component identified by namespace, name.
// If it is not found, nil, nil is returned
func (c *Client) GetComponent(namespace, name string) (*Component, error) {
	labelSelect := fmt.Sprintf("%v=%v,%v=%v", "component.onepanel.io/part-of", "onepanel", "component.onepanel.io/name", name)

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
		return nil, fmt.Errorf("non-unique result found for GetComponent %v,%v", namespace, name)
	}

	serviceItem := serviceList.Items[0]
	componentName := serviceItem.Labels["component.onepanel.io/name"]
	componentURL, err := c.generateComponentURL(namespace, componentName)
	if err != nil {
		return nil, err
	}

	component := &Component{
		Name: componentName,
		URL:  componentURL,
	}

	return component, nil
}

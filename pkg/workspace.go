package v1

import (
	"github.com/onepanelio/core/pkg/util"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	networking "istio.io/api/networking/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

func parseServicePorts(template string) (servicePorts []*corev1.ServicePort, err error) {
	if err = yaml.UnmarshalStrict([]byte(template), &servicePorts); err != nil {
		return
	}

	return
}

func parseHTTPRoutes(template string) (HTTPRoutes []*networking.HTTPRoute, err error) {
	if err = yaml.UnmarshalStrict([]byte(template), &HTTPRoutes); err != nil {
		return
	}

	return
}

func parseVolumeClaims(template string) (persistentVolumeClaims []*corev1.PersistentVolumeClaim, err error) {
	if err = yaml.UnmarshalStrict([]byte(template), &persistentVolumeClaims); err != nil {
		return
	}

	return
}

func parseContainers(template string) (containers []*corev1.Container, err error) {
	if err = yaml.UnmarshalStrict([]byte(template), &containers); err != nil {
		return
	}

	return
}

func (c *Client) CreateWorkspace(namespace, parametersTemplate, containersTemplate, portsTemplate, routesTemplate, volumeClaimsTemplate string) (err error) {
	_, err = parseContainers(containersTemplate)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Error":     err.Error(),
		}).Error("Invalid Workspace Containers template.")
		return util.NewUserError(codes.InvalidArgument, err.Error())
	}

	return
}

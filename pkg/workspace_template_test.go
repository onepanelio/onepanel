package v1

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	portsManifest = `
- name: http
  port: 80
  protocol: TCP
  targetPort: 80
- name: https
  port: 443
  protocol: TCP
  targetPort: 443
`
	routesManifest = `
- match:
  - uri:
      prefix: /
  route:
  - destination:
      port:
        number: 80
- match:
  - uri:
      prefix: /
  route:
  - destination:
      port:
        number: 443
`

	volumeClaimsManifest = `
- metadata:
    name: data
  spec:
    accessModes: ["ReadWriteOnce"]
    storageClassName: default
`
	containersManifest = `
- name: http
  image: nginxdemos/hello
  ports:
  - containerPort: 80
    name: http
  volumeMounts:
  - name: data
    mountPath: /data
- name: https
  image: nginxdemos/hello
  ports:
  - containerPort: 443
    name: http
  volumeMounts:
  - name: data
    mountPath: /data
`
	workspaceTemplateManifest = WorkspaceTemplate{
		VolumeClaimsManifest: volumeClaimsManifest,
		ContainersManifest:   containersManifest,
		PortsManifest:        portsManifest,
		RoutesManifest:       routesManifest,
	}
)

func TestParseServicePorts(t *testing.T) {
	servicePorts, err := parsePorts(portsManifest)
	assert.Nil(t, err)
	assert.NotEmpty(t, servicePorts)
	assert.Equal(t, servicePorts[0].Name, "http")
	assert.Equal(t, servicePorts[1].Name, "https")
}

func TestParseServicePortsInvalid(t *testing.T) {
	template := `
- name: http
  port: 80
  invalid: TCP
  targetPort: 80
`

	_, err := parsePorts(template)
	assert.NotNil(t, err)
}

func TestParseHTTPRoutes(t *testing.T) {
	httpRoutes, err := parseRoutes(routesManifest)
	assert.Nil(t, err)
	assert.NotEmpty(t, httpRoutes)
	assert.Equal(t, httpRoutes[0].Match[0].Uri.GetPrefix(), "/")
	assert.Equal(t, httpRoutes[1].Route[0].Destination.Port.GetNumber(), uint32(443))
}

func TestParseHTTPRoutesInvalid(t *testing.T) {
	template := `
- match:
  - invalid:
      prefix: /
  route:
  - destination:
      port:
        number: 80
`

	_, err := parseRoutes(template)
	assert.NotNil(t, err)
}

func TestParseVolumeClaims(t *testing.T) {
	volumeClaims, err := parseVolumeClaims(volumeClaimsManifest)
	assert.Nil(t, err)
	assert.NotEmpty(t, volumeClaims)
}

func TestParseVolumeClaimsInvalid(t *testing.T) {
	template := `
- metadata:
    name: data
  invalid:
    accessModes: ["ReadWriteOnce"]
    storageClassName: default
- metadata:
    name: db
`

	_, err := parseVolumeClaims(template)
	assert.NotNil(t, err)
}

func TestParseContainers(t *testing.T) {
	containers, err := parseContainers(containersManifest)
	assert.Nil(t, err)
	assert.NotEmpty(t, containers)
	assert.Equal(t, containers[0].Ports[0].ContainerPort, int32(80))
	assert.Equal(t, containers[1].Ports[0].ContainerPort, int32(443))
}

func TestParseContainersInvalid(t *testing.T) {
	template := `
- name: https
  image: nginxdemos/hello
  invalid:
  - containerPort: 443
    name: http
  volumeMounts:
  - name: data
    mountPath: /data
`

	_, err := parseContainers(template)
	assert.NotNil(t, err)
}

func TestCreateWorkspaceTemplate(t *testing.T) {
	c := NewTestClient(fakeSystemSecret, fakeSystemConfigMap)

	if err := c.CreateWorkspaceTemplate("rush", workspaceTemplateManifest); err != nil {
		t.Error(err)
	}
}

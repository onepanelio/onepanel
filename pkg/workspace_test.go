package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseServicePorts(t *testing.T) {
	template := `
- name: http
  port: 80
  protocol: TCP
  targetPort: 80
- name: https
  port: 443
  protocol: TCP
  targetPort: 443
`

	servicePorts, err := parseServicePorts(template)
	assert.Nil(t, err)
	assert.NotEmpty(t, servicePorts)
	assert.Equal(t, servicePorts[0].Name, "http")
	assert.Equal(t, servicePorts[1].Name, "https")
}

func TestParseHTTPRoutes(t *testing.T) {
	template := `
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

	httpRoutes, err := parseHTTPRoutes(template)
	assert.Nil(t, err)
	assert.NotEmpty(t, httpRoutes)
	assert.Equal(t, httpRoutes[0].Match[0].Uri.GetPrefix(), "/")
	assert.Equal(t, httpRoutes[1].Route[0].Destination.Port.GetNumber(), uint32(443))
}

func TestParseVolumeClaims(t *testing.T) {
	template := `
- metadata:
    name: data
  spec:
    accessModes: ["ReadWriteOnce"]
    storageClassName: default
- metadata:
    name: db
`

	volumeClaims, err := parseVolumeClaims(template)
	assert.Nil(t, err)
	assert.NotEmpty(t, volumeClaims)
}

func TestParseContainers(t *testing.T) {
	template := `
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

	containers, err := parseContainers(template)
	assert.Nil(t, err)
	assert.NotEmpty(t, containers)
	assert.Equal(t, containers[0].Ports[0].ContainerPort, int32(80))
	assert.Equal(t, containers[1].Ports[0].ContainerPort, int32(443))
}

func TestParseContainersInvalid(t *testing.T) {
	template := `
- name: https
  image: nginxdemos/hello
  pors:
  - containerPort: 443
    name: http
  volumeMounts:
  - name: data
    mountPath: /data
`

	containers, err := parseContainers(template)
	assert.NotNil(t, err)
	assert.Empty(t, containers)
}

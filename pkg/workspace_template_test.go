package v1

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	workspaceSpecManifest = `
containers:
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
ports:
- name: http
  port: 80
  protocol: TCP
  targetPort: 80
- name: https
  port: 443
  protocol: TCP
  targetPort: 443
routes:
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
	workspaceTemplate = WorkspaceTemplate{
		Manifest: workspaceSpecManifest,
	}
)

func TestParseWorkspaceSpec(t *testing.T) {
	workspaceSpec, err := parseWorkspaceSpec(workspaceSpecManifest)
	assert.Nil(t, err)
	assert.NotEmpty(t, workspaceSpec)
	assert.Equal(t, workspaceSpec.Ports[0].Name, "http")
	assert.Equal(t, workspaceSpec.Ports[1].Name, "https")
	assert.Equal(t, workspaceSpec.Routes[0].Match[0].Uri.GetPrefix(), "/")
	assert.Equal(t, workspaceSpec.Routes[1].Route[0].Destination.Port.GetNumber(), uint32(443))
	assert.Equal(t, workspaceSpec.Containers[0].Ports[0].ContainerPort, int32(80))
	assert.Equal(t, workspaceSpec.Containers[1].Ports[0].ContainerPort, int32(443))
}

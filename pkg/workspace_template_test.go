package v1

import (
	"testing"
)

var (
	workspaceSpecManifest = `
arguments:
  parameters:
    - name: description
      value: description
      type: textarea.textarea
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

//func TestParseServicePorts(t *testing.T) {
//	servicePorts, err := parsePorts(portsManifest)
//	assert.Nil(t, err)
//	assert.NotEmpty(t, servicePorts)
//	assert.Equal(t, servicePorts[0].Name, "http")
//	assert.Equal(t, servicePorts[1].Name, "https")
//}
//
//func TestParseServicePortsInvalid(t *testing.T) {
//	template := `
//- name: http
//  port: 80
//  invalid: TCP
//  targetPort: 80
//`
//
//	_, err := parsePorts(template)
//	assert.NotNil(t, err)
//}
//
//func TestParseHTTPRoutes(t *testing.T) {
//	httpRoutes, err := parseRoutes(routesManifest)
//	assert.Nil(t, err)
//	assert.NotEmpty(t, httpRoutes)
//	assert.Equal(t, httpRoutes[0].Match[0].Uri.GetPrefix(), "/")
//	assert.Equal(t, httpRoutes[1].Route[0].Destination.Port.GetNumber(), uint32(443))
//}
//
//func TestParseHTTPRoutesInvalid(t *testing.T) {
//	template := `
//- match:
//  - invalid:
//      prefix: /
//  route:
//  - destination:
//      port:
//        number: 80
//`
//
//	_, err := parseRoutes(template)
//	assert.NotNil(t, err)
//}
//
//func TestParseVolumeClaims(t *testing.T) {
//	volumeClaims, err := parseVolumeClaims(volumeClaimsManifest)
//	assert.Nil(t, err)
//	assert.NotEmpty(t, volumeClaims)
//}
//
//func TestParseVolumeClaimsInvalid(t *testing.T) {
//	template := `
//- metadata:
//    name: data
//  invalid:
//    accessModes: ["ReadWriteOnce"]
//    storageClassName: default
//- metadata:
//    name: db
//`
//
//	_, err := parseVolumeClaims(template)
//	assert.NotNil(t, err)
//}
//
//func TestParseContainers(t *testing.T) {
//	containers, err := parseContainers(containersManifest)
//	assert.Nil(t, err)
//	assert.NotEmpty(t, containers)
//	assert.Equal(t, containers[0].Ports[0].ContainerPort, int32(80))
//	assert.Equal(t, containers[1].Ports[0].ContainerPort, int32(443))
//}
//
//func TestParseContainersInvalid(t *testing.T) {
//	template := `
//- name: https
//  image: nginxdemos/hello
//  invalid:
//  - containerPort: 443
//    name: http
//  volumeMounts:
//  - name: data
//    mountPath: /data
//`
//
//	_, err := parseContainers(template)
//	assert.NotNil(t, err)
//}

func TestCreateWorkspaceTemplate(t *testing.T) {
	c := NewTestClient(mockSystemSecret, mockSystemConfigMap)

	if err := c.CreateWorkspaceTemplate("rush", workspaceTemplate); err != nil {
		t.Error(err)
	}
}

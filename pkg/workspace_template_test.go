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

	jupyterLabWorkspaceManifest = `# Docker containers that are part of the Workspace
containers:
- name: jupyterlab-tensorflow
  image: jupyter/tensorflow-notebook
  command: [start.sh, jupyter]
  workingDir: /data
  env:
    - name: tornado
      value: "{ 'headers': { 'Content-Security-Policy': \"frame-ancestors * 'self'\" }  }"
    - name: GRANT_SUDO
      value: 1
    - name: CHOWN_EXTRA
      value: '/data'
    - name: CHOWN_EXTRA_OPTS
      value: '-R'
  securityContext:
    runAsUser: 0
    allowPrivilegeEscalation: false
  args:
    - lab
    - --LabApp.token=''
    - --LabApp.allow_remote_access=True
    - --LabApp.allow_origin="*"
    - --LabApp.disable_check_xsrf=True
    - --LabApp.trust_xheaders=True
    - --LabApp.tornado_settings=$(tornado)
    - --NotebookApp.notebook_dir='/data'
  ports:
  - containerPort: 8888
    name: jupyterlab
  # Volumes to be mounted in this container
  # Onepanel will automatically create these volumes and mount them to the container
  volumeMounts:
  - name: data
    mountPath: /data
# Ports that need to be exposed
ports:
- name: jupyterlab
  port: 80
  protocol: TCP
  targetPort: 8888
# Routes that will map to ports
routes:
- match:
  - uri:
      prefix: /
  route:
  - destination:
      port:
        number: 80
# DAG Workflow to be executed once a Workspace action completes
# postExecutionWorkflow:
#   entrypoint: main
#   templates:
#   - name: main
#     dag:
#        tasks:
#        - name: slack-notify
#          template: slack-notify
#   - name: slack-notify
#     container:
#       image: technosophos/slack-notify
#       args:
#       - SLACK_USERNAME=onepanel SLACK_TITLE="Your workspace is ready" SLACK_ICON=https://www.gravatar.com/avatar/5c4478592fe00878f62f0027be59c1bd SLACK_MESSAGE="Your workspace is now running" ./slack-notify
#       command:
#       - sh
#       - -c
`
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

// testClientCreateWorkspaceTemplateNew creates a new workspace template
func testClientCreateWorkspaceTemplateNew(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	namespace := "onepanel"

	wt := &WorkspaceTemplate{
		Name:     "test",
		Manifest: jupyterLabWorkspaceManifest,
	}

	_, err := c.CreateWorkspaceTemplate(namespace, wt)

	assert.Nil(t, err)
}

// testClientCreateWorkspaceTemplateDuplicateName attempts to create a workspace template for a name that already exists
// this should error
func testClientCreateWorkspaceTemplateDuplicateName(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	namespace := "onepanel"

	wt := &WorkspaceTemplate{
		Name:     "test",
		Manifest: jupyterLabWorkspaceManifest,
	}

	_, err := c.CreateWorkspaceTemplate(namespace, wt)
	_, err = c.CreateWorkspaceTemplate(namespace, wt)

	assert.NotNil(t, err)
}

// testClientCreateWorkspaceTemplateArchivedName attempts to create a workspace template for a name that has been archived
// this should work
func testClientCreateWorkspaceTemplateArchivedName(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	namespace := "onepanel"

	wt := &WorkspaceTemplate{
		Name:     "test",
		Manifest: jupyterLabWorkspaceManifest,
	}

	wtCreated, err := c.CreateWorkspaceTemplate(namespace, wt)
	_, err = c.ArchiveWorkspaceTemplate(namespace, wtCreated.UID)
	_, err = c.CreateWorkspaceTemplate(namespace, wt)

	assert.Nil(t, err)
}

// TestClient_CreateWorkspaceTemplate tests creating a workspace template
func TestClient_CreateWorkspaceTemplate(t *testing.T) {
	testClientCreateWorkspaceTemplateNew(t)
	testClientCreateWorkspaceTemplateDuplicateName(t)
	testClientCreateWorkspaceTemplateArchivedName(t)
}

func testClientArchiveWorkspaceTemplateSuccess(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	namespace := "onepanel"

	wt := &WorkspaceTemplate{
		Name:     "test",
		Manifest: jupyterLabWorkspaceManifest,
	}

	testTemplate, _ := c.CreateWorkspaceTemplate(namespace, wt)
	archived, err := c.ArchiveWorkspaceTemplate(namespace, testTemplate.UID)

	assert.Nil(t, err)
	assert.True(t, archived)
}

// testClientArchiveWorkspaceTemplateNotFound tests the case where you try to archive a non-existing workspace template
func testClientArchiveWorkspaceTemplateNotFound(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	namespace := "onepanel"

	archived, err := c.ArchiveWorkspaceTemplate(namespace, "not-found")

	assert.NotNil(t, err)
	assert.False(t, archived)
}

// TestClient_ArchiveWorkspaceTemplate tests archiving a workspace template
func TestClient_ArchiveWorkspaceTemplate(t *testing.T) {
	testClientArchiveWorkspaceTemplateSuccess(t)
	testClientArchiveWorkspaceTemplateNotFound(t)

	// TODO we need more tests here to make sure the related resources are cleaned up, including workspaces and workflow templates
}

// testClientListWorkspaceTemplatesEmpty tests listing workspace templates when there are none
func testClientListWorkspaceTemplatesEmpty(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	templates, err := c.ListWorkspaceTemplates("onepanel", nil)

	assert.Nil(t, err)
	assert.Empty(t, templates)
}

// testClientListWorkspaceTemplatesNotEmpty tests listing workspaces when there are records that are
// archived and not. It should only list the non-archived ones
func testClientListWorkspaceTemplatesNotEmpty(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	namespace := "onepanel"

	wt := &WorkspaceTemplate{
		Name:     "test",
		Manifest: jupyterLabWorkspaceManifest,
	}

	testTemplate, _ := c.CreateWorkspaceTemplate(namespace, wt)
	c.ArchiveWorkspaceTemplate(namespace, testTemplate.UID)

	wt2 := &WorkspaceTemplate{
		Name:     "test2",
		Manifest: jupyterLabWorkspaceManifest,
	}

	wt2, _ = c.CreateWorkspaceTemplate(namespace, wt2)

	templates, err := c.ListWorkspaceTemplates(namespace, nil)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(templates))
	assert.Equal(t, wt2.UID, templates[0].UID)
}

// TestClient_ListWorkspaceTemplates tests listing workspace templates
func TestClient_ListWorkspaceTemplates(t *testing.T) {
	testClientListWorkspaceTemplatesEmpty(t)
	testClientListWorkspaceTemplatesNotEmpty(t)
}

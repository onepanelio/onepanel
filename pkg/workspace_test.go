package v1

import (
	"github.com/lib/pq"
	"github.com/onepanelio/core/pkg/util"
	"github.com/onepanelio/core/pkg/util/ptr"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"testing"
	"time"
)

func testWorkspaceStatusToFieldMapLaunching(t *testing.T) {
	start := time.Now().UTC()
	fm := workspaceStatusToFieldMap(&WorkspaceStatus{Phase: WorkspaceLaunching})

	assert.Equal(t, fm["phase"], WorkspaceLaunching)
	assert.Equal(t, fm["paused_at"], pq.NullTime{})

	started := fm["started_at"].(time.Time)

	assert.True(t, started.Nanosecond() > start.Nanosecond())
}

func testWorkspaceStatusToFieldMapPausing(t *testing.T) {
	start := time.Now().UTC()
	fm := workspaceStatusToFieldMap(&WorkspaceStatus{Phase: WorkspacePausing})

	assert.Equal(t, fm["phase"], WorkspacePausing)
	assert.Equal(t, fm["started_at"], pq.NullTime{})

	paused := fm["paused_at"].(time.Time)

	assert.True(t, paused.Nanosecond() > start.Nanosecond())
}

func testWorkspaceStatusToFieldMapUpdating(t *testing.T) {
	start := time.Now().UTC()
	fm := workspaceStatusToFieldMap(&WorkspaceStatus{Phase: WorkspaceUpdating})

	assert.Equal(t, fm["phase"], WorkspaceUpdating)
	assert.Equal(t, fm["paused_at"], pq.NullTime{})

	updated := fm["updated_at"].(time.Time)

	assert.True(t, updated.Nanosecond() > start.Nanosecond())
}

func testWorkspaceStatusToFieldMapTerminating(t *testing.T) {
	start := time.Now().UTC()
	fm := workspaceStatusToFieldMap(&WorkspaceStatus{Phase: WorkspaceTerminating})

	assert.Equal(t, fm["phase"], WorkspaceTerminating)
	assert.Equal(t, fm["paused_at"], pq.NullTime{})
	assert.Equal(t, fm["started_at"], pq.NullTime{})

	terminated := fm["terminated_at"].(time.Time)

	assert.True(t, terminated.Nanosecond() > start.Nanosecond())
}

func Test_WorkspaceStatusToFieldMap(t *testing.T) {
	testWorkspaceStatusToFieldMapLaunching(t)
	testWorkspaceStatusToFieldMapPausing(t)
	testWorkspaceStatusToFieldMapUpdating(t)
	testWorkspaceStatusToFieldMapTerminating(t)
}

// testClientPrivateCreateWorkspaceNoWorkflowTemplate makes sure we get an error when there is no workflow template for the workspace
func testClientPrivateCreateWorkspaceNoWorkflowTemplate(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	namespace := "onepanel"

	workspace := &Workspace{
		Name: "test",
		WorkspaceTemplate: &WorkspaceTemplate{
			WorkflowTemplate: &WorkflowTemplate{
				UID:     "not-exist",
				Version: 1,
			},
		},
	}
	workspace.GenerateUID("test")

	_, err := c.createWorkspace(namespace, []byte(""), workspace)

	userErr, ok := err.(*util.UserError)
	assert.True(t, ok)
	assert.Equal(t, userErr.Code, codes.NotFound)
}

// testClientPrivateCreateWorkspaceSuccess tests creating a workspace successfully
func testClientPrivateCreateWorkspaceSuccess(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	namespace := "onepanel"

	workspaceTemplate := &WorkspaceTemplate{
		Name:     "test",
		Manifest: jupyterLabWorkspaceManifest,
	}

	workspaceTemplate, _ = c.CreateWorkspaceTemplate(namespace, workspaceTemplate)

	workspace := &Workspace{
		Name:              "test2",
		WorkspaceTemplate: workspaceTemplate,
		Parameters: []Parameter{
			{
				Name:  "workflow-execution-name",
				Value: ptr.String("test2"),
			},
		},
	}
	workspace.GenerateUID("test")

	_, err := c.createWorkspace(namespace, []byte("{}"), workspace)

	assert.Nil(t, err)
}

func TestClient_createWorkspace(t *testing.T) {
	testClientPrivateCreateWorkspaceNoWorkflowTemplate(t)
	testClientPrivateCreateWorkspaceSuccess(t)
}

func TestClient_CreateWorkspace(t *testing.T) {

}

func TestClient_ArchiveWorkspace(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	namespace := "onepanel"

	workspaceTemplate := &WorkspaceTemplate{
		Name:     "test",
		Manifest: jupyterLabWorkspaceManifest,
	}

	workspaceTemplate, _ = c.CreateWorkspaceTemplate(namespace, workspaceTemplate)

	workspace := &Workspace{
		Name:              "test2",
		WorkspaceTemplate: workspaceTemplate,
		Parameters: []Parameter{
			{
				Name:  "workflow-execution-name",
				Value: ptr.String("test2"),
			},
		},
	}
	workspace.GenerateUID("test")

	createdWorkspace, _ := c.createWorkspace(namespace, []byte("[]"), workspace)

	params := []Parameter{
		{
			Name:  "workflow-execution-name",
			Value: ptr.String("test3"),
		},
	}
	err := c.ArchiveWorkspace(namespace, createdWorkspace.UID, params...)

	assert.Nil(t, err)
}

// TestClient_ListWorkspacesByTemplateID tests listing workspaces by the template id
func TestClient_ListWorkspacesByTemplateID(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	namespace := "onepanel"

	wt := &WorkspaceTemplate{
		Name:     "test",
		Manifest: jupyterLabWorkspaceManifest,
	}

	testTemplate, _ := c.CreateWorkspaceTemplate(namespace, wt)
	workspace := &Workspace{
		Name:              "test",
		WorkspaceTemplate: testTemplate,
		Parameters: []Parameter{
			{
				Name:  "workflow-execution-name",
				Value: ptr.String("test"),
			},
		},
	}
	workspace.GenerateUID("test")

	c.createWorkspace(namespace, []byte("[]"), workspace)

	wt2 := &WorkspaceTemplate{
		Name:     "test2",
		Manifest: jupyterLabWorkspaceManifest,
	}

	testTemplate2, _ := c.CreateWorkspaceTemplate(namespace, wt2)
	workspace2 := &Workspace{
		Name:              "test2",
		WorkspaceTemplate: testTemplate2,
		Parameters: []Parameter{
			{
				Name:  "workflow-execution-name",
				Value: ptr.String("test2"),
			},
		},
	}
	workspace2.GenerateUID("test2")

	c.createWorkspace(namespace, []byte("[]"), workspace2)

	workspaces, err := c.ListWorkspacesByTemplateID(namespace, testTemplate.ID)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(workspaces))

	params := []Parameter{
		{
			Name:  "workflow-execution-name",
			Value: ptr.String("test3"),
		},
	}
	c.ArchiveWorkspace(namespace, testTemplate.UID, params...)

	workspaces, err = c.ListWorkspacesByTemplateID(namespace, testTemplate.ID)
	assert.Nil(t, err)
	assert.True(t, workspaces[0].Status.Phase == WorkspaceTerminating)
}

package v1

import (
	"github.com/asaskevich/govalidator"
	"github.com/stretchr/testify/assert"
	"testing"
)

func assertWorkspaceNameInvalid(t *testing.T, name string) {
	ws := Workspace{
		UID:  "test",
		Name: name,
	}

	valid, _ := govalidator.ValidateStruct(ws)

	assert.False(t, valid)
}

func assertWorkspaceNameValid(t *testing.T, name string) {
	ws := Workspace{
		UID:  "test",
		Name: name,
	}

	valid, _ := govalidator.ValidateStruct(ws)

	assert.True(t, valid)
}

func TestWorkspaceNameValidation_RegexValid(t *testing.T) {
	assertWorkspaceNameInvalid(t, "600s")

	assertWorkspaceNameValid(t, "test-5")
	assertWorkspaceNameValid(t, "test 5")
	assertWorkspaceNameValid(t, "TEst 5")
	assertWorkspaceNameValid(t, "CVAT")
	assertWorkspaceNameValid(t, "My CVAT Workspace")
	assertWorkspaceNameValid(t, "CVAT Workspace 1")
}

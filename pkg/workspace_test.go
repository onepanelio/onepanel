package v1

import (
	"github.com/asaskevich/govalidator"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWorkspaceNameValidation_Regex(t *testing.T) {
	bad := Workspace{
		UID:  "test",
		Name: "600s",
	}

	valid, err := govalidator.ValidateStruct(bad)

	assert.False(t, valid)

	good := Workspace{
		UID:  "test",
		Name: "test-5",
	}

	valid, err = govalidator.ValidateStruct(good)
	if err != nil {
		t.Error(err)
	}

	assert.True(t, valid)
}

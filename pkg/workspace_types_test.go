package v1

import (
	"github.com/asaskevich/govalidator"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWorkspaceNameValidation_Regex_Invalid(t *testing.T) {
	bad := Workspace{
		UID:  "test",
		Name: "600s",
	}

	valid, _ := govalidator.ValidateStruct(bad)
	assert.False(t, valid)
}

func TestWorkspaceNameValidation_RegexValid(t *testing.T) {
	good := Workspace{
		UID:  "test",
		Name: "test-5",
	}

	valid, _ := govalidator.ValidateStruct(good)
	assert.True(t, valid)

	goodSpaces := Workspace{
		UID:  "test",
		Name: "test 5",
	}

	valid, _ = govalidator.ValidateStruct(goodSpaces)
	assert.True(t, valid)

	goodUppercase := Workspace{
		UID:  "test",
		Name: "TEst 5",
	}

	valid, _ = govalidator.ValidateStruct(goodUppercase)
	assert.True(t, valid)
}

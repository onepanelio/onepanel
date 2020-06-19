package sqlutil

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_FormatSelect_Columns_NoData(t *testing.T) {
	result := FormatColumnSelect([]string{})

	assert.Equal(t, len(result), 0)
}

func Test_FormatSelect_Columns(t *testing.T) {
	result := FormatColumnSelect([]string{"name", "uid"})

	assert.Equal(t, len(result), 2)

	for _, item := range result {
		if item != "name" && item != "uid" {
			t.Error("item not in possible list")
		}
	}
}

func Test_FormatSelect_Alias(t *testing.T) {
	result := FormatColumnSelect([]string{"name", "uid"}, "u")

	assert.Equal(t, len(result), 2)

	for _, item := range result {
		if item != "u.name" && item != "u.uid" {
			t.Error("item not in possible list")
		}
	}
}

func Test_FormatSelect_AliasDestination(t *testing.T) {
	result := FormatColumnSelect([]string{"name", "uid"}, "u", "user")

	assert.Equal(t, len(result), 2)

	for _, item := range result {
		if item != `u.name "user.name"` && item != `u.uid "user.uid"` {
			t.Error("item not in possible list")
		}
	}
}

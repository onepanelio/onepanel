package converter

import (
	"github.com/onepanelio/core/api"
)

func APIKeyValueToLabel(apiKeyValues []*api.KeyValue) map[string]string {
	result := make(map[string]string)
	if apiKeyValues == nil {
		return result
	}

	for _, entry := range apiKeyValues {
		result[entry.Key] = entry.Value
	}

	return result
}

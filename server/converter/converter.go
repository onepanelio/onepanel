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

func MappingToKeyValue(mapping map[string]string) []*api.KeyValue {
	keyValues := make([]*api.KeyValue, 0)

	for key, value := range mapping {
		keyValues = append(keyValues, &api.KeyValue{
			Key:   key,
			Value: value,
		})
	}

	return keyValues
}

package mapping

import (
	"fmt"
	"gopkg.in/yaml.v2"
)

type Mapping map[interface{}]interface{}

func PluckKeys(input map[interface{}]interface{}) []interface{} {
	result := make([]interface{}, 0)

	for key := range input {
		result = append(result, key)
	}

	return result
}

func PluckKeysStr(input map[string]string) []*string {
	result := make([]*string, 0)

	for key := range input {
		result = append(result, &key)
	}

	return result
}

func New() Mapping {
	return make(Mapping)
}

func NewFromYamlBytes(data []byte) (Mapping, error) {
	mapping := New()

	if err := yaml.Unmarshal(data, mapping); err != nil {
		return nil, err
	}

	return mapping, nil
}

func NewFromYamlString(data string) (Mapping, error) {
	return NewFromYamlBytes([]byte(data))
}

// Marshals the mapping using yaml
// if mapping is nil, an empty byte array is returned.
func (m Mapping) ToYamlBytes() ([]byte, error) {
	if m == nil {
		return make([]byte, 0), nil
	}

	return yaml.Marshal(m)
}

// Removes all the empty values in the mapping, going in depth.
func (m Mapping) PruneEmpty() {
	if m == nil {
		return
	}

	deleteEmptyValuesMapping(m)
}

// Attempts the get the Mapping under key.
// If it is not a mapping, an error is returned.
// If it does not exist, it is created and returned.
func (m Mapping) GetChildMap(key interface{}) (Mapping, error) {
	subMapping, ok := m[key]
	if !ok {
		subMapping = New()
		m[key] = subMapping
	}

	asMap, ok := subMapping.(Mapping)
	if !ok {
		return nil, fmt.Errorf("%v is not a Mapping", key)
	}

	return asMap, nil
}

// Returns the number of keys in the map
func deleteEmptyValuesMapping(mapping Mapping) int {
	keys := 0
	for key, value := range mapping {
		keys++
		valueAsMapping, ok := value.(Mapping)
		if ok {
			if deleteEmptyValuesMapping(valueAsMapping) == 0 {
				delete(mapping, key)
			}
		}

		valueAsArray, ok := value.([]interface{})
		if ok {
			deleteEmptyValuesArray(valueAsArray)
		}

		valueAsString, ok := value.(string)
		if ok && valueAsString == "" {
			delete(mapping, key)
		}
	}

	return keys
}

// Returns the number of items in the array.
func deleteEmptyValuesArray(values []interface{}) int {
	count := 0
	for _, value := range values {
		count++

		valueAsMapping, ok := value.(Mapping)
		if ok {
			deleteEmptyValuesMapping(valueAsMapping)
		}

		valueAsArray, ok := value.([]interface{})
		if ok {
			deleteEmptyValuesArray(valueAsArray)
		}
	}

	return count
}

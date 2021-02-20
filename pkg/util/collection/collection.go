package collection

import (
	"reflect"
)

// RepeatSymbol returns symbol <count> times, with <separator> between each one.
// if symbol = ?, separator = , and count = 5
// this returns: "?,?,?,?,?"
func RepeatSymbol(count int, symbol, separator string) string {
	result := ""
	for i := 0; i < count; i++ {
		if i != 0 {
			result += separator
		}

		result += symbol
	}

	return result
}

// RemoveBlanks goes through the data, assumed to be an array or map of some kind,
// and removes any data that is a nil or zero value. Maps with no keys are also removed.
//
// Note that this will not check the data again. So if you have the following
// parent: {
//   child: {}
// }
//
// The result will be
// parent: {}
//
// it will not go through it again and remove parent.
func RemoveBlanks(data interface{}) {
	if mapping, ok := data.(map[string]interface{}); ok {
		keysToDelete := make([]string, 0)
		for key, v := range mapping {
			rv := reflect.ValueOf(v)
			if v == nil || rv.IsZero() {
				keysToDelete = append(keysToDelete, key)
			} else if vMap, vMapOk := v.(map[string]interface{}); vMapOk && len(vMap) == 0 {
				keysToDelete = append(keysToDelete, key)
			} else {
				RemoveBlanks(v)
			}
		}

		for _, keyToDelete := range keysToDelete {
			delete(mapping, keyToDelete)
		}
	} else if list, ok := data.([]interface{}); ok {
		for _, listItem := range list {
			RemoveBlanks(listItem)
		}
	}
}

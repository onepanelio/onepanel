package v1

import (
	"fmt"
	"strings"
	"time"
)

// Label represents a database-backed label row.
type Label struct {
	ID         uint64
	CreatedAt  time.Time `db:"created_at"`
	Key        string
	Value      string
	Resource   string
	ResourceID uint64 `db:"resource_id"`
}

// LabelsToMapping converts Label structs to a map of key:value
func LabelsToMapping(labels ...*Label) map[string]string {
	result := make(map[string]string)

	for _, label := range labels {
		result[label.Key] = label.Value
	}

	return result
}

// LabelsFromString parses a string into labels
// Format: key=<key>,value=<value>&key2=<key2>,value2=<value2>
func LabelsFromString(value string) (labels []*Label, err error) {
	labels = make([]*Label, 0)

	labelParts := strings.Split(value, "&")
	if len(labelParts) == 0 {
		return
	}

	for _, part := range labelParts {
		newLabel, err := LabelFromString(part)
		if err != nil {
			return labels, err
		}
		if newLabel == nil {
			continue
		}

		labels = append(labels, newLabel)
	}

	return
}

// LabelFromString converts a parses into a label
// Format: key=<key>,value=<value>
func LabelFromString(value string) (label *Label, err error) {
	parts := strings.Split(value, ",")
	if len(parts) != 2 {
		return nil, fmt.Errorf("label does not have two parts, key/value")
	}

	label = &Label{}

	first := parts[0]
	firstItems := strings.Split(first, "=")
	if len(firstItems) != 2 {
		return nil, fmt.Errorf(`incorrectly formatted label "%v"`, first)
	}

	if firstItems[0] == "key" {
		label.Key = firstItems[1]
	} else if firstItems[0] == "value" {
		label.Value = firstItems[1]
	}

	second := parts[1]
	secondItems := strings.Split(second, "=")
	if len(secondItems) != 2 {
		return nil, fmt.Errorf(`incorrectly formatted label "%v"`, second)
	}

	if secondItems[0] == "key" {
		label.Key = secondItems[1]
	} else if secondItems[0] == "value" {
		label.Value = secondItems[1]
	}

	return label, nil
}

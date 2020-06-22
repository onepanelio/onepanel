package v1

import "time"

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

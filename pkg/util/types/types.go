package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type Labels map[string]string

// Unmarshal unmarshal's the json in j to v, as in json.Unmarshal.
func (l *Labels) Unmarshal(v interface{}) error {
	if len(*l) == 0 {
		*l = make(map[string]string)
	}
	v = l
	return nil
}

// Value returns j as a value.  This does a validating unmarshal into another
// RawMessage.  If j is invalid json, it returns an error.
func (l Labels) Value() (driver.Value, error) {
	return json.Marshal(l)
}

// Scan stores the src in *j.  No validation is done.
func (l *Labels) Scan(src interface{}) error {
	var source []byte
	switch t := src.(type) {
	case string:
		source = []byte(t)
	case []byte:
		if len(t) == 0 {
			source = []byte("{}")
		} else {
			source = t
		}
	case nil:
		*l = make(map[string]string)
	default:
		// TODO rename to JSONLabels
		return errors.New("Incompatible type for Labels")
	}

	return json.Unmarshal(source, l)
}

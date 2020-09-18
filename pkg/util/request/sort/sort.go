package sort

import (
	"fmt"
	"strings"
)

// Order represents a sorting order such as created_at, desc
type Order struct {
	Property  string
	Direction string
}

// Criteria represents the sorting criteria for a list of resources
type Criteria struct {
	Properties []Order
}

// New parses the properties, represented as comma delimited fields, into a Criteria struct
// The first part is the properties, the second part is the delimiter used between properties. If none is provided,
// a semi-colon (;) is used.
// Each property is assumed to be of the form: propertyName,desc;propertyName2;asc
// example: createdAt,desc;name,asc
func New(parts ...string) (*Criteria, error) {
	if len(parts) == 0 {
		return nil, fmt.Errorf("no properties provided to create a Criteria")
	}

	separator := ";"
	if len(parts) > 1 {
		separator = parts[1]
	}

	criteria := &Criteria{
		Properties: make([]Order, 0),
	}

	if parts[0] == "" {
		return criteria, nil
	}

	items := strings.Split(parts[0], separator)

	for _, item := range items {
		parts := strings.Split(item, ",")
		if len(parts) != 2 {
			return nil, fmt.Errorf("badly formatted sort: '%v'", item)
		}

		direction := strings.ToLower(parts[1])

		if direction != "asc" && direction != "desc" {
			return nil, fmt.Errorf("unknown sort '%v'", parts[1])
		}

		newSort := Order{
			Property:  parts[0],
			Direction: direction,
		}

		criteria.Properties = append(criteria.Properties, newSort)
	}

	return criteria, nil
}

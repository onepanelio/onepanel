package sql

import (
	"fmt"
)

// FormatColumnSelect returns a list of column names to be used in a SQL Select modified with optional alias and destination.
//
// aliasAndDestination supports two arguments, an alias followed by a destination. Any arguments after are ignored.
//
// If an alias is provided, each column is prefixed with it. Otherwise the columns are returned as is.
//
// If a destination is provided, each column will be assigned to it. Otherwise there is no adjustment.
//
// Example - alias, no destination.
// Input: ([id, name], "w")
// Output: [w.id, w.name]
//
// Example - with alias, destination
// Input: ([id, name], "w", "workflow")
// Output: [w.id "workflow.id", w.name "workflow.name"]
func FormatColumnSelect(columns []string, aliasAndDestination ...string) []string {
	results := make([]string, 0)

	alias := ""
	destination := ""

	if len(aliasAndDestination) > 0 {
		alias = aliasAndDestination[0]
	}

	if len(aliasAndDestination) > 1 {
		destination = aliasAndDestination[1]
	}

	for _, str := range columns {
		result := str

		if alias != "" {
			result = alias + "." + result
		}

		if destination != "" {
			result += fmt.Sprintf(` "%v.%v"`, destination, str)
		}
		results = append(results, result)
	}

	return results
}

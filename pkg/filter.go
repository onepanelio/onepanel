package v1

import sq "github.com/Masterminds/squirrel"

// LabelFilter represents a filter that has labels
type LabelFilter interface {
	// GetLabels returns the labels to filter by. These are assumed to be ANDed together.
	GetLabels() []*Label
}

// ApplyLabelSelectQuery returns a query builder that adds where statements to filter by labels in the filter, if there are any
// labelSelector is the database column that has the labels, such as "we.labels" for workflowExecutions aliased by "we".
func ApplyLabelSelectQuery(labelSelector string, sb sq.SelectBuilder, filter LabelFilter) (sq.SelectBuilder, error) {
	labels := filter.GetLabels()

	if len(labels) == 0 {
		return sb, nil
	}

	labelsJSON, err := LabelsToJSONString(labels)
	if err != nil {
		return sb, err
	}

	sb = sb.Where("%v @> ?", labelSelector, labelsJSON)

	return sb, nil
}

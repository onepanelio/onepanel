package request

import (
	"github.com/Masterminds/squirrel"
	"github.com/onepanelio/core/pkg/util/request/pagination"
	"github.com/onepanelio/core/pkg/util/request/sort"
)

// Request creates a new resource request with criteria to pagination, filter, and sort the results.
type Request struct {
	Pagination *pagination.PaginationRequest
	Filter     interface{}
	Sort       *sort.Criteria
}

// HasSorting returns true if there are any sorting criteria in the request
func (r *Request) HasSorting() bool {
	return r != nil &&
		r.Sort != nil &&
		len(r.Sort.Properties) > 0
}

// HasFilter returns true if there is any filtering criteria in the request
func (r *Request) HasFilter() bool {
	return r != nil &&
		r.Filter != nil
}

// ApplyPaginationToSelect applies the pagination to the selectBuilder, if there is a pagination.
func (r *Request) ApplyPaginationToSelect(sb *squirrel.SelectBuilder) *squirrel.SelectBuilder {
	if r == nil || r.Pagination == nil {
		return sb
	}

	return r.Pagination.ApplyToSelect(sb)
}

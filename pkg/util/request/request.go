package request

import (
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
	return r.Sort != nil && len(r.Sort.Properties) > 0
}

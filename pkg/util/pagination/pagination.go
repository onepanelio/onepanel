package pagination

import (
	"github.com/Masterminds/squirrel"
	"math"
)

type PaginationRequest struct {
	Page     uint64
	PageSize uint64
}

func NewRequest(page, pageSize int32) PaginationRequest {
	if page == 0 {
		page = 1
	}

	if pageSize == 0 {
		pageSize = 15
	}

	return PaginationRequest{
		Page:     uint64(page),
		PageSize: uint64(pageSize),
	}
}

// Start creates a new PaginationRequest that starts at the first page.
// You can provide an optional pageSize argument. If none is provided, 15 is used.
// All arguments apart from the first one are ignored.
func Start(pageSize ...int32) *PaginationRequest {
	if len(pageSize) > 0 {
		pr := NewRequest(1, pageSize[0])
		return &pr
	}

	pr := NewRequest(1, 15)

	return &pr
}

func (pr *PaginationRequest) Offset() uint64 {
	// start at page 1.
	return (pr.Page - 1) * pr.PageSize
}

func (pr *PaginationRequest) CalculatePages(count int) int32 {
	return int32(math.Ceil(float64(count) / float64(pr.PageSize)))
}

func (pr *PaginationRequest) ApplyToSelect(sb *squirrel.SelectBuilder) *squirrel.SelectBuilder {
	if pr == nil {
		return sb
	}

	result := sb.Limit(pr.PageSize).
		Offset(pr.Offset())

	return &result
}

// Advance returns a new pagination request with the page incremented by 1
func (pr *PaginationRequest) Advance() *PaginationRequest {
	return &PaginationRequest{
		Page:     pr.Page + 1,
		PageSize: pr.PageSize,
	}
}

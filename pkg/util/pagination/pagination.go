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

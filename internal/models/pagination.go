package models

type PaginationParams struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

type PaginationMetadata struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	Total      int  `json:"total"`
	TotalPages int  `json:"totalPages"`
	HasNext    bool `json:"hasNext"`
	HasPrev    bool `json:"hasPrev"`
}

func NewPaginationParams(page, limit int) PaginationParams {
	if page < 1 {
		page = 1
	}

	if limit < 1 {
		limit = 20
	}

	if limit > 100 {
		limit = 100
	}

	return PaginationParams{
		Page:  page,
		Limit: limit,
	}
}

func (p PaginationParams) CalculateMetadata(total int) PaginationMetadata {
	// Si es menor a 1 toma 1
	totalPages := max(1, (total+p.Limit-1)/p.Limit)

	return PaginationMetadata{
		Page:       p.Page,
		Limit:      p.Limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    p.Page < totalPages,
		HasPrev:    p.Page > 1,
	}
}

func (p PaginationParams) GetOffset() int {
	return (p.Page - 1) * p.Limit
}

func (p PaginationParams) GetEndIndex(total int) int {
	// Si end es mayor a total entonces end == total
	end := min(total, p.GetOffset()+p.Limit)

	return end

}

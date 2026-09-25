// Package pagination provides request pagination helpers
package pagination

import (
	"math"
	"net/http"
	"strconv"
)

// Params holds pagination parameters
type Params struct {
	Limit  int
	Offset int
	Page   int
}

// Response wraps paginated results
type Response struct {
	Data       interface{} `json:"data"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
	HasNext    bool        `json:"has_next"`
	HasPrev    bool        `json:"has_prev"`
}

const (
	DefaultLimit = 25
	MaxLimit     = 1000
)

// ParseFromRequest extracts pagination params from URL query
func ParseFromRequest(r *http.Request) Params {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	pageStr := r.URL.Query().Get("page")

	limit := DefaultLimit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	// Clamp limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}
	if offset < 0 {
		offset = 0
	}

	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil {
			page = p
		}
	}
	if page < 1 {
		page = 1
	}

	// If page is specified, convert to offset
	if pageStr != "" {
		offset = (page - 1) * limit
	} else if offsetStr != "" {
		page = (offset / limit) + 1
	}

	return Params{
		Limit:  limit,
		Offset: offset,
		Page:   page,
	}
}

// Paginate wraps results with pagination metadata
func Paginate(data interface{}, total int, page int, pageSize int) Response {
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if totalPages < 1 {
		totalPages = 1
	}

	return Response{
		Data:       data,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

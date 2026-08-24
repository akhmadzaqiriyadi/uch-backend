package pagination

import (
	"math"
	"net/http"
	"strconv"
)

type Params struct {
	Page    int    `json:"page"`
	PerPage int    `json:"per_page"`
	SortBy  string `json:"sort_by,omitempty"`
	Order   string `json:"order,omitempty"`
}

type Meta struct {
	CurrentPage int  `json:"current_page"`
	PerPage     int  `json:"per_page"`
	TotalItems  int  `json:"total_items"`
	TotalPages  int  `json:"total_pages"`
	HasNext     bool `json:"has_next"`
	HasPrev     bool `json:"has_prev"`
}

func FromRequest(r *http.Request) Params {
	query := r.URL.Query()

	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(query.Get("per_page"))
	if perPage < 1 {
		perPage = 10
	}
	if perPage > 100 {
		perPage = 100 // Prevent fetching thousands of records
	}

	sortBy := query.Get("sort_by")
	if sortBy == "" {
		sortBy = "created_at"
	}

	order := query.Get("order")
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	return Params{
		Page:    page,
		PerPage: perPage,
		SortBy:  sortBy,
		Order:   order,
	}
}

func (p Params) Offset() int {
	return (p.Page - 1) * p.PerPage
}

func (p Params) Limit() int {
	return p.PerPage
}

func BuildMeta(p Params, totalItems int) Meta {
	totalPages := int(math.Ceil(float64(totalItems) / float64(p.PerPage)))
	if totalPages == 0 {
		totalPages = 1
	}

	return Meta{
		CurrentPage: p.Page,
		PerPage:     p.PerPage,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
		HasNext:     p.Page < totalPages,
		HasPrev:     p.Page > 1,
	}
}

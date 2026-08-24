package pagination

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromRequest_Defaults(t *testing.T) {
	req := httptest.NewRequest("GET", "/users", nil)
	params := FromRequest(req)

	assert.Equal(t, 1, params.Page)
	assert.Equal(t, 10, params.PerPage)
	assert.Equal(t, "created_at", params.SortBy)
	assert.Equal(t, "desc", params.Order)
	assert.Equal(t, 0, params.Offset())
	assert.Equal(t, 10, params.Limit())
}

func TestFromRequest_CustomValidValues(t *testing.T) {
	req := httptest.NewRequest("GET", "/users?page=3&per_page=25&sort_by=name&order=asc", nil)
	params := FromRequest(req)

	assert.Equal(t, 3, params.Page)
	assert.Equal(t, 25, params.PerPage)
	assert.Equal(t, "name", params.SortBy)
	assert.Equal(t, "asc", params.Order)
	assert.Equal(t, 50, params.Offset())
	assert.Equal(t, 25, params.Limit())
}

func TestFromRequest_BoundsAndInvalidValues(t *testing.T) {
	req := httptest.NewRequest("GET", "/users?page=-5&per_page=500&sort_by=&order=invalid", nil)
	params := FromRequest(req)

	assert.Equal(t, 1, params.Page)
	assert.Equal(t, 100, params.PerPage) // Clamped to 100 max
	assert.Equal(t, "created_at", params.SortBy)
	assert.Equal(t, "desc", params.Order) // Fallback to desc
}

func TestBuildMeta(t *testing.T) {
	tests := []struct {
		name       string
		params     Params
		totalItems int
		wantMeta   Meta
	}{
		{
			name:       "first page of multiple",
			params:     Params{Page: 1, PerPage: 10},
			totalItems: 45,
			wantMeta: Meta{
				CurrentPage: 1,
				PerPage:     10,
				TotalItems:  45,
				TotalPages:  5,
				HasNext:     true,
				HasPrev:     false,
			},
		},
		{
			name:       "middle page",
			params:     Params{Page: 3, PerPage: 10},
			totalItems: 45,
			wantMeta: Meta{
				CurrentPage: 3,
				PerPage:     10,
				TotalItems:  45,
				TotalPages:  5,
				HasNext:     true,
				HasPrev:     true,
			},
		},
		{
			name:       "last page",
			params:     Params{Page: 5, PerPage: 10},
			totalItems: 45,
			wantMeta: Meta{
				CurrentPage: 5,
				PerPage:     10,
				TotalItems:  45,
				TotalPages:  5,
				HasNext:     false,
				HasPrev:     true,
			},
		},
		{
			name:       "empty results",
			params:     Params{Page: 1, PerPage: 10},
			totalItems: 0,
			wantMeta: Meta{
				CurrentPage: 1,
				PerPage:     10,
				TotalItems:  0,
				TotalPages:  1,
				HasNext:     false,
				HasPrev:     false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildMeta(tt.params, tt.totalItems)
			assert.Equal(t, tt.wantMeta, got)
		})
	}
}

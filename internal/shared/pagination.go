package shared

import (
	"net/http"
	"strconv"
)

const (
	DefaultPage    = 1
	DefaultPerPage = 20
	MaxPerPage     = 100
)

type PaginationParams struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

type PaginationMeta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

func ParsePagination(r *http.Request) PaginationParams {
	page := queryInt(r, "page", DefaultPage)
	perPage := queryInt(r, "per_page", DefaultPerPage)

	if page < 1 {
		page = DefaultPage
	}
	if perPage < 1 {
		perPage = DefaultPerPage
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}

	return PaginationParams{
		Page:    page,
		PerPage: perPage,
	}
}

func (p PaginationParams) Offset() int {
	return (p.Page - 1) * p.PerPage
}

func (p PaginationParams) Limit() int {
	return p.PerPage
}

func NewPaginationMeta(params PaginationParams, total int64) PaginationMeta {
	totalPages := int(total) / params.PerPage
	if int(total)%params.PerPage > 0 {
		totalPages++
	}
	if totalPages < 1 {
		totalPages = 1
	}

	return PaginationMeta{
		Page:       params.Page,
		PerPage:    params.PerPage,
		Total:      total,
		TotalPages: totalPages,
	}
}

func queryInt(r *http.Request, key string, fallback int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return fallback
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return i
}

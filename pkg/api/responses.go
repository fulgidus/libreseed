package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// SuccessResponse represents a standardized success response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// Meta contains response metadata (pagination, etc.)
type Meta struct {
	Page       int `json:"page,omitempty"`
	PerPage    int `json:"per_page,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
	TotalItems int `json:"total_items,omitempty"`
}

// PaginationParams represents pagination query parameters
type PaginationParams struct {
	Page    int
	PerPage int
}

// DefaultPagination returns default pagination parameters
func DefaultPagination() PaginationParams {
	return PaginationParams{
		Page:    1,
		PerPage: 20,
	}
}

// ParsePagination extracts pagination parameters from request
func ParsePagination(r *http.Request) PaginationParams {
	params := DefaultPagination()

	if page := r.URL.Query().Get("page"); page != "" {
		if p, err := parseInt(page); err == nil && p > 0 {
			params.Page = p
		}
	}

	if perPage := r.URL.Query().Get("per_page"); perPage != "" {
		if pp, err := parseInt(perPage); err == nil && pp > 0 && pp <= 100 {
			params.PerPage = pp
		}
	}

	return params
}

// CalculateMeta calculates pagination metadata
func CalculateMeta(page, perPage, totalItems int) *Meta {
	totalPages := (totalItems + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}

	return &Meta{
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
		TotalItems: totalItems,
	}
}

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}

// WriteSuccess writes a standardized success response
func WriteSuccess(w http.ResponseWriter, data interface{}) error {
	return WriteJSON(w, http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

// WriteSuccessWithMeta writes a success response with pagination metadata
func WriteSuccessWithMeta(w http.ResponseWriter, data interface{}, meta *Meta) error {
	return WriteJSON(w, http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// WriteCreated writes a 201 Created response
func WriteCreated(w http.ResponseWriter, data interface{}) error {
	return WriteJSON(w, http.StatusCreated, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

// WriteNoContent writes a 204 No Content response
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Helper function to parse integers
func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

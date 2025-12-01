package api

import (
	"encoding/json"
	"net/http"
)

// ErrorCode represents standardized API error codes
type ErrorCode string

const (
	ErrCodeBadRequest          ErrorCode = "BAD_REQUEST"
	ErrCodeUnauthorized        ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden           ErrorCode = "FORBIDDEN"
	ErrCodeNotFound            ErrorCode = "NOT_FOUND"
	ErrCodeConflict            ErrorCode = "CONFLICT"
	ErrCodeValidation          ErrorCode = "VALIDATION_ERROR"
	ErrCodeRateLimit           ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrCodeInternalServer      ErrorCode = "INTERNAL_SERVER_ERROR"
	ErrCodeServiceUnavailable  ErrorCode = "SERVICE_UNAVAILABLE"
)

// ErrorResponse represents a standardized API error response
type ErrorResponse struct {
	Error   ErrorDetail `json:"error"`
	Request RequestInfo `json:"request,omitempty"`
}

// ErrorDetail contains error information
type ErrorDetail struct {
	Code    ErrorCode              `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// RequestInfo contains request context for debugging
type RequestInfo struct {
	Method    string `json:"method,omitempty"`
	Path      string `json:"path,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// APIError represents an internal API error with HTTP status
type APIError struct {
	Code       ErrorCode
	Message    string
	StatusCode int
	Details    map[string]interface{}
}

// Error implements the error interface
func (e *APIError) Error() string {
	return e.Message
}

// NewAPIError creates a new API error
func NewAPIError(code ErrorCode, message string, statusCode int) *APIError {
	return &APIError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Details:    make(map[string]interface{}),
	}
}

// WithDetail adds a detail field to the error
func (e *APIError) WithDetail(key string, value interface{}) *APIError {
	e.Details[key] = value
	return e
}

// Common error constructors

// BadRequest creates a 400 Bad Request error
func BadRequest(message string) *APIError {
	return NewAPIError(ErrCodeBadRequest, message, http.StatusBadRequest)
}

// Unauthorized creates a 401 Unauthorized error
func Unauthorized(message string) *APIError {
	return NewAPIError(ErrCodeUnauthorized, message, http.StatusUnauthorized)
}

// Forbidden creates a 403 Forbidden error
func Forbidden(message string) *APIError {
	return NewAPIError(ErrCodeForbidden, message, http.StatusForbidden)
}

// NotFound creates a 404 Not Found error
func NotFound(resource string) *APIError {
	return NewAPIError(ErrCodeNotFound, resource+" not found", http.StatusNotFound)
}

// Conflict creates a 409 Conflict error
func Conflict(message string) *APIError {
	return NewAPIError(ErrCodeConflict, message, http.StatusConflict)
}

// ValidationError creates a 422 Validation Error
func ValidationError(message string) *APIError {
	return NewAPIError(ErrCodeValidation, message, http.StatusUnprocessableEntity)
}

// RateLimitExceeded creates a 429 Rate Limit Exceeded error
func RateLimitExceeded(message string) *APIError {
	return NewAPIError(ErrCodeRateLimit, message, http.StatusTooManyRequests)
}

// InternalServerError creates a 500 Internal Server Error
func InternalServerError(message string) *APIError {
	return NewAPIError(ErrCodeInternalServer, message, http.StatusInternalServerError)
}

// ServiceUnavailable creates a 503 Service Unavailable error
func ServiceUnavailable(message string) *APIError {
	return NewAPIError(ErrCodeServiceUnavailable, message, http.StatusServiceUnavailable)
}

// WriteError writes an API error response
func WriteError(w http.ResponseWriter, r *http.Request, err *APIError) {
	response := ErrorResponse{
		Error: ErrorDetail{
			Code:    err.Code,
			Message: err.Message,
			Details: err.Details,
		},
		Request: RequestInfo{
			Method:    r.Method,
			Path:      r.URL.Path,
			RequestID: GetRequestID(r),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.StatusCode)
	json.NewEncoder(w).Encode(response)
}

// GetRequestID extracts request ID from context (set by middleware)
func GetRequestID(r *http.Request) string {
	if id := r.Context().Value("request_id"); id != nil {
		if idStr, ok := id.(string); ok {
			return idStr
		}
	}
	return ""
}

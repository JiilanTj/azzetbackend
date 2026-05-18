package shared

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// Error code constants
const (
	CodeBadRequest        = "BAD_REQUEST"
	CodeUnauthorized      = "UNAUTHORIZED"
	CodeForbidden         = "FORBIDDEN"
	CodeNotFound          = "NOT_FOUND"
	CodeConflict          = "CONFLICT"
	CodeValidation        = "VALIDATION_ERROR"
	CodeInternal          = "INTERNAL_ERROR"
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)

// APIResponse is the standard success response envelope.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

// APIError is the standard error response envelope.
type APIError struct {
	Code      string        `json:"code"`
	Message   string        `json:"message"`
	Domain    string        `json:"domain"`
	Details   []FieldError  `json:"details,omitempty"`
	RequestID string        `json:"request_id,omitempty"`
	Timestamp string        `json:"timestamp"`
}

// FieldError represents a single field validation error.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ErrorResponse wraps APIError in the standard envelope.
type ErrorResponse struct {
	Success bool     `json:"success"`
	Error   APIError `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

// Success writes a success response with data.
func Success(w http.ResponseWriter, status int, data interface{}) {
	writeJSON(w, status, APIResponse{
		Success: true,
		Data:    data,
	})
}

// SuccessMessage writes a success response with a message wrapped in data.
func SuccessMessage(w http.ResponseWriter, status int, msg string, data interface{}) {
	payload := map[string]interface{}{
		"message": msg,
	}
	if data != nil {
		payload["data"] = data
	}
	writeJSON(w, status, APIResponse{
		Success: true,
		Data:    payload,
	})
}

// Paginated writes a success response with pagination meta.
func Paginated(w http.ResponseWriter, status int, data interface{}, meta PaginationMeta) {
	writeJSON(w, status, APIResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// Error writes a structured error response.
func Error(w http.ResponseWriter, r *http.Request, status int, code, domain, message string) {
	reqID := middleware.GetReqID(r.Context())
	writeJSON(w, status, ErrorResponse{
		Success: false,
		Error: APIError{
			Code:      code,
			Message:   message,
			Domain:    domain,
			RequestID: reqID,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// ValidationError writes a validation error response with field details.
func ValidationError(w http.ResponseWriter, r *http.Request, domain, message string, details []FieldError) {
	reqID := middleware.GetReqID(r.Context())
	writeJSON(w, http.StatusBadRequest, ErrorResponse{
		Success: false,
		Error: APIError{
			Code:      CodeValidation,
			Message:   message,
			Domain:    domain,
			Details:   details,
			RequestID: reqID,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// Unauthorized writes a 401 error response.
func Unauthorized(w http.ResponseWriter, r *http.Request, domain, message string) {
	Error(w, r, http.StatusUnauthorized, CodeUnauthorized, domain, message)
}

// Forbidden writes a 403 error response.
func Forbidden(w http.ResponseWriter, r *http.Request, domain, message string) {
	Error(w, r, http.StatusForbidden, CodeForbidden, domain, message)
}

// NotFound writes a 404 error response.
func NotFound(w http.ResponseWriter, r *http.Request, domain, message string) {
	Error(w, r, http.StatusNotFound, CodeNotFound, domain, message)
}

// InternalError writes a 500 error response.
func InternalError(w http.ResponseWriter, r *http.Request, domain, message string) {
	Error(w, r, http.StatusInternalServerError, CodeInternal, domain, message)
}

// Conflict writes a 409 error response.
func Conflict(w http.ResponseWriter, r *http.Request, domain, message string) {
	Error(w, r, http.StatusConflict, CodeConflict, domain, message)
}

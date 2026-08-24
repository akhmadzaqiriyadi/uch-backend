package response

import (
	"encoding/json"
	"net/http"
	"time"
)

// APIResponse represents the standard unified JSON response envelope
type APIResponse struct {
	Success    bool      `json:"success"`
	StatusCode int       `json:"status_code"`
	ErrorCode  string    `json:"error_code,omitempty"`
	Message    string    `json:"message"`
	Timestamp  time.Time `json:"timestamp"`
	Data       any       `json:"data,omitempty"`
	Errors     any       `json:"errors,omitempty"`
}

// JSON sends a raw HTTP response with the APIResponse envelope
func JSON(w http.ResponseWriter, statusCode int, payload APIResponse) {
	if payload.Timestamp.IsZero() {
		payload.Timestamp = time.Now().UTC()
	}
	payload.StatusCode = statusCode

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

// ==========================================
// Success Response Helpers (2xx)
// ==========================================

// Success sends a generic successful response
func Success(w http.ResponseWriter, statusCode int, message string, data any) {
	JSON(w, statusCode, APIResponse{
		Success:   true,
		Message:   message,
		Timestamp: time.Now().UTC(),
		Data:      data,
	})
}

// OK sends a 200 OK response
func OK(w http.ResponseWriter, message string, data any) {
	Success(w, http.StatusOK, message, data)
}

// Created sends a 201 Created response
func Created(w http.ResponseWriter, message string, data any) {
	Success(w, http.StatusCreated, message, data)
}

// NoContent sends a 204 No Content response
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// ==========================================
// Client Error Response Helpers (4xx)
// ==========================================

// Error sends a generic error response
func Error(w http.ResponseWriter, statusCode int, errorCode, message string, errorDetails any) {
	JSON(w, statusCode, APIResponse{
		Success:   false,
		ErrorCode: errorCode,
		Message:   message,
		Timestamp: time.Now().UTC(),
		Errors:    errorDetails,
	})
}

// BadRequest sends a 400 Bad Request error
func BadRequest(w http.ResponseWriter, message string, errorDetails any) {
	if message == "" {
		message = "Invalid request payload or parameters"
	}
	Error(w, http.StatusBadRequest, "BAD_REQUEST", message, errorDetails)
}

// Unauthorized sends a 401 Unauthorized error
func Unauthorized(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Authentication required or token expired"
	}
	Error(w, http.StatusUnauthorized, "UNAUTHORIZED", message, nil)
}

// Forbidden sends a 403 Forbidden error
func Forbidden(w http.ResponseWriter, message string, details any) {
	if message == "" {
		message = "You do not have permission to access this resource"
	}
	Error(w, http.StatusForbidden, "FORBIDDEN", message, details)
}

// NotFound sends a 404 Not Found error
func NotFound(w http.ResponseWriter, message string) {
	if message == "" {
		message = "The requested resource was not found"
	}
	Error(w, http.StatusNotFound, "NOT_FOUND", message, nil)
}

// Conflict sends a 409 Conflict error (e.g. duplicate email/username)
func Conflict(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Resource conflict: record already exists"
	}
	Error(w, http.StatusConflict, "CONFLICT", message, nil)
}

// UnprocessableEntity sends a 422 Validation Error
func UnprocessableEntity(w http.ResponseWriter, message string, validationErrors any) {
	if message == "" {
		message = "Validation failed on one or more fields"
	}
	Error(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", message, validationErrors)
}

// TooManyRequests sends a 429 Rate Limit error
func TooManyRequests(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Rate limit exceeded. Please try again later."
	}
	Error(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", message, nil)
}

// ==========================================
// Server Error Response Helpers (5xx)
// ==========================================

// InternalServerError sends a 500 Internal Server Error
func InternalServerError(w http.ResponseWriter, message string, errorDetails any) {
	if message == "" {
		message = "An internal server error occurred. Please try again later."
	}
	Error(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", message, errorDetails)
}

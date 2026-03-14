package errors

import (
	"encoding/json"
	"log"
	"net/http"
)

type ErrorCode string

const (
	ErrCodeBadRequest          ErrorCode = "BAD_REQUEST"
	ErrCodeNotFound            ErrorCode = "NOT_FOUND"
	ErrCodeInternalServerError ErrorCode = "INTERNAL_SERVER_ERROR"
	ErrCodeServiceUnavailable  ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeTooManyRequests     ErrorCode = "TOO_MANY_REQUESTS"
	ErrCodeInvalidInput        ErrorCode = "INVALID_INPUT"
	ErrCodeExternalAPIError    ErrorCode = "EXTERNAL_API_ERROR"
	ErrCodeCacheError          ErrorCode = "CACHE_ERROR"
	ErrCodeDatabaseError       ErrorCode = "DATABASE_ERROR"
	ErrCodeTimeout             ErrorCode = "TIMEOUT"
)

type APIError struct {
	Success bool        `json:"success"`
	Message string      `json:"error"`
	Code    ErrorCode   `json:"code,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

func NewAPIError(code ErrorCode, message string) APIError {
	return APIError{
		Success: false,
		Message: message,
		Code:    code,
	}
}

func BadRequest(message string) APIError {
	return NewAPIError(ErrCodeBadRequest, message)
}

func NotFound(resource string) APIError {
	return NewAPIError(ErrCodeNotFound, resource+" not found")
}

func InternalError(message string) APIError {
	return NewAPIError(ErrCodeInternalServerError, message)
}

func ServiceUnavailable(message string) APIError {
	return NewAPIError(ErrCodeServiceUnavailable, message)
}

func TooManyRequests(message string) APIError {
	return NewAPIError(ErrCodeTooManyRequests, message)
}

func InvalidInput(field string, message string) APIError {
	return APIError{
		Success: false,
		Message: message,
		Code:    ErrCodeInvalidInput,
		Details: map[string]string{"field": field},
	}
}

func ExternalAPIError(service string, err error) APIError {
	return APIError{
		Success: false,
		Message: "External API error: " + err.Error(),
		Code:    ErrCodeExternalAPIError,
		Details: map[string]string{"service": service},
	}
}

func DatabaseError(err error) APIError {
	return APIError{
		Success: false,
		Message: "Database error: " + err.Error(),
		Code:    ErrCodeDatabaseError,
	}
}

func CacheError(err error) APIError {
	return APIError{
		Success: false,
		Message: "Cache error: " + err.Error(),
		Code:    ErrCodeCacheError,
	}
}

func TimeoutError(operation string) APIError {
	return NewAPIError(ErrCodeTimeout, "Operation timed out: "+operation)
}

func (e APIError) Error() string {
	return e.Message
}

func (e APIError) WithDetails(details interface{}) APIError {
	e.Details = details
	return e
}

func RespondWithError(w http.ResponseWriter, r *http.Request, err APIError) {
	statusCode := http.StatusInternalServerError
	switch err.Code {
	case ErrCodeBadRequest, ErrCodeInvalidInput:
		statusCode = http.StatusBadRequest
	case ErrCodeNotFound:
		statusCode = http.StatusNotFound
	case ErrCodeTooManyRequests:
		statusCode = http.StatusTooManyRequests
	case ErrCodeServiceUnavailable:
		statusCode = http.StatusServiceUnavailable
	case ErrCodeTimeout:
		statusCode = http.StatusRequestTimeout
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(err)
}

func RespondWithSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    data,
	})
}

func HandleServiceError(w http.ResponseWriter, r *http.Request, service string, err error) {
	log.Printf("[ERROR] %s service error: %v", service, err)
	RespondWithError(w, r, ExternalAPIError(service, err))
}

func HandlePanic(w http.ResponseWriter, r *http.Request) {
	if rec := recover(); rec != nil {
		log.Printf("[PANIC] %v", rec)
		RespondWithError(w, r, InternalError("An unexpected error occurred"))
	}
}

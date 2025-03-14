package apperrors

import (
	"fmt"
	"net/http"
)

// AppError represents an application error
type AppError struct {
	Code    int
	Message string
	Err     error
}

// Error returns the error message
func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(message string, err error) *AppError {
	return &AppError{
		Code:    http.StatusNotFound,
		Message: message,
		Err:     err,
	}
}

// NewBadRequestError creates a new bad request error
func NewBadRequestError(message string, err error) *AppError {
	return &AppError{
		Code:    http.StatusBadRequest,
		Message: message,
		Err:     err,
	}
}

// NewInternalServerError creates a new internal server error
func NewInternalServerError(message string, err error) *AppError {
	return &AppError{
		Code:    http.StatusInternalServerError,
		Message: message,
		Err:     err,
	}
}

// NewUnauthorizedError creates a new unauthorized error
func NewUnauthorizedError(message string, err error) *AppError {
	return &AppError{
		Code:    http.StatusUnauthorized,
		Message: message,
		Err:     err,
	}
}

// Response represents an error response
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// NewErrorResponse creates a new error response
func NewErrorResponse(message string, err error) Response {
	return Response{
		Success: false,
		Message: message,
		Error:   err.Error(),
	}
}

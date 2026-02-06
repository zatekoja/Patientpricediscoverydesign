package errors

import (
	"fmt"
)

// ErrorType represents different types of errors in the system
type ErrorType string

const (
	// ErrorTypeNotFound indicates a resource was not found
	ErrorTypeNotFound ErrorType = "NOT_FOUND"
	
	// ErrorTypeValidation indicates a validation error
	ErrorTypeValidation ErrorType = "VALIDATION"
	
	// ErrorTypeConflict indicates a conflict with existing data
	ErrorTypeConflict ErrorType = "CONFLICT"
	
	// ErrorTypeUnauthorized indicates unauthorized access
	ErrorTypeUnauthorized ErrorType = "UNAUTHORIZED"
	
	// ErrorTypeInternal indicates an internal server error
	ErrorTypeInternal ErrorType = "INTERNAL"
	
	// ErrorTypeExternal indicates an error from external service
	ErrorTypeExternal ErrorType = "EXTERNAL"
)

// AppError represents an application error
type AppError struct {
	Type    ErrorType
	Message string
	Err     error
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap implements the unwrap interface
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(message string) *AppError {
	return &AppError{
		Type:    ErrorTypeNotFound,
		Message: message,
	}
}

// NewValidationError creates a new validation error
func NewValidationError(message string) *AppError {
	return &AppError{
		Type:    ErrorTypeValidation,
		Message: message,
	}
}

// NewConflictError creates a new conflict error
func NewConflictError(message string) *AppError {
	return &AppError{
		Type:    ErrorTypeConflict,
		Message: message,
	}
}

// NewUnauthorizedError creates a new unauthorized error
func NewUnauthorizedError(message string) *AppError {
	return &AppError{
		Type:    ErrorTypeUnauthorized,
		Message: message,
	}
}

// NewInternalError creates a new internal error
func NewInternalError(message string, err error) *AppError {
	return &AppError{
		Type:    ErrorTypeInternal,
		Message: message,
		Err:     err,
	}
}

// NewExternalError creates a new external service error
func NewExternalError(message string, err error) *AppError {
	return &AppError{
		Type:    ErrorTypeExternal,
		Message: message,
		Err:     err,
	}
}

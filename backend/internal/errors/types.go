package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ErrorType represents the category of error
type ErrorType string

const (
	ErrorTypeValidation   ErrorType = "validation"
	ErrorTypeAuth         ErrorType = "authentication"
	ErrorTypePermission   ErrorType = "permission"
	ErrorTypeNotFound     ErrorType = "not_found"
	ErrorTypeConflict     ErrorType = "conflict"
	ErrorTypeRateLimit    ErrorType = "rate_limit"
	ErrorTypeQuotaExceeded ErrorType = "quota_exceeded"
	ErrorTypeInternal     ErrorType = "internal"
	ErrorTypeExternal     ErrorType = "external"
	ErrorTypeBadRequest   ErrorType = "bad_request"
)

// AppError represents a structured application error
type AppError struct {
	Type      ErrorType   `json:"type"`
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Details   string      `json:"details,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Context   interface{} `json:"context,omitempty"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	return e.Message
}

// HTTPStatus returns the appropriate HTTP status code for the error
func (e *AppError) HTTPStatus() int {
	if e.Code >= 100 && e.Code < 600 {
		return e.Code
	}

	// Default mappings based on error type
	switch e.Type {
	case ErrorTypeValidation, ErrorTypeBadRequest:
		return http.StatusBadRequest
	case ErrorTypeAuth:
		return http.StatusUnauthorized
	case ErrorTypePermission:
		return http.StatusForbidden
	case ErrorTypeNotFound:
		return http.StatusNotFound
	case ErrorTypeConflict:
		return http.StatusConflict
	case ErrorTypeRateLimit:
		return http.StatusTooManyRequests
	case ErrorTypeQuotaExceeded:
		return http.StatusPaymentRequired // 402
	case ErrorTypeExternal:
		return http.StatusBadGateway
	case ErrorTypeInternal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// ToJSON converts the error to JSON format
func (e *AppError) ToJSON() []byte {
	data, _ := json.Marshal(e)
	return data
}

// WithContext adds context to the error
func (e *AppError) WithContext(ctx interface{}) *AppError {
	e.Context = ctx
	return e
}

// WithDetails adds additional details to the error
func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

// WithRequestID adds a request ID to the error
func (e *AppError) WithRequestID(requestID string) *AppError {
	e.RequestID = requestID
	return e
}

// NewError creates a new AppError
func NewError(errorType ErrorType, message string) *AppError {
	return &AppError{
		Type:      errorType,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}
}

// NewErrorWithCode creates a new AppError with a specific HTTP code
func NewErrorWithCode(errorType ErrorType, code int, message string) *AppError {
	return &AppError{
		Type:      errorType,
		Code:      code,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}
}

// Predefined error constructors

// NewValidationError creates a validation error
func NewValidationError(message string, details ...string) *AppError {
	err := &AppError{
		Type:      ErrorTypeValidation,
		Code:      http.StatusBadRequest,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}
	if len(details) > 0 {
		err.Details = details[0]
	}
	return err
}

// NewAuthError creates an authentication error
func NewAuthError(message string) *AppError {
	return &AppError{
		Type:      ErrorTypeAuth,
		Code:      http.StatusUnauthorized,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}
}

// NewPermissionError creates a permission error
func NewPermissionError(message string) *AppError {
	return &AppError{
		Type:      ErrorTypePermission,
		Code:      http.StatusForbidden,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resource string) *AppError {
	return &AppError{
		Type:      ErrorTypeNotFound,
		Code:      http.StatusNotFound,
		Message:   fmt.Sprintf("%s not found", resource),
		Timestamp: time.Now().UTC(),
	}
}

// NewConflictError creates a conflict error
func NewConflictError(message string) *AppError {
	return &AppError{
		Type:      ErrorTypeConflict,
		Code:      http.StatusConflict,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}
}

// NewRateLimitError creates a rate limit error
func NewRateLimitError(message string) *AppError {
	return &AppError{
		Type:      ErrorTypeRateLimit,
		Code:      http.StatusTooManyRequests,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}
}

// NewQuotaExceededError creates a quota exceeded error
func NewQuotaExceededError(message string) *AppError {
	return &AppError{
		Type:      ErrorTypeQuotaExceeded,
		Code:      http.StatusPaymentRequired,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}
}

// NewInternalError creates an internal server error
func NewInternalError(message string, details ...string) *AppError {
	err := &AppError{
		Type:      ErrorTypeInternal,
		Code:      http.StatusInternalServerError,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}
	if len(details) > 0 {
		err.Details = details[0]
	}
	return err
}

// NewExternalError creates an external service error
func NewExternalError(service, message string) *AppError {
	return &AppError{
		Type:      ErrorTypeExternal,
		Code:      http.StatusBadGateway,
		Message:   fmt.Sprintf("External service error (%s): %s", service, message),
		Timestamp: time.Now().UTC(),
	}
}

// NewBadRequestError creates a bad request error
func NewBadRequestError(message string) *AppError {
	return &AppError{
		Type:      ErrorTypeBadRequest,
		Code:      http.StatusBadRequest,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}
}

// ErrorResponse represents the JSON error response format
type ErrorResponse struct {
	Error     *AppError `json:"error"`
	Success   bool      `json:"success"`
	Timestamp time.Time `json:"timestamp"`
}

// HandleError is a Gin middleware for handling application errors
func HandleError() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.Next()

		// Check if there are any errors
		if len(c.Errors) == 0 {
			return
		}

		// Get the last error
		err := c.Errors.Last()

		var appError *AppError

		// Convert to AppError if needed
		switch e := err.Err.(type) {
		case *AppError:
			appError = e
		default:
			appError = NewInternalError("Internal server error", e.Error())
		}

		// Add request ID if available
		if requestID := c.GetString("request_id"); requestID != "" {
			appError = appError.WithRequestID(requestID)
		}

		// Create response
		response := ErrorResponse{
			Error:     appError,
			Success:   false,
			Timestamp: time.Now().UTC(),
		}

		// Set appropriate status code and return JSON
		c.JSON(appError.HTTPStatus(), response)
		c.Abort()
	})
}

// AbortWithError aborts the request with an AppError
func AbortWithError(c *gin.Context, appError *AppError) {
	// Add request ID if available
	if requestID := c.GetString("request_id"); requestID != "" {
		appError = appError.WithRequestID(requestID)
	}

	response := ErrorResponse{
		Error:     appError,
		Success:   false,
		Timestamp: time.Now().UTC(),
	}

	c.JSON(appError.HTTPStatus(), response)
	c.Abort()
}

// SuccessResponse represents a successful API response
type SuccessResponse struct {
	Data      interface{} `json:"data,omitempty"`
	Message   string      `json:"message,omitempty"`
	Success   bool        `json:"success"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

// RespondSuccess sends a success response
func RespondSuccess(c *gin.Context, data interface{}, message ...string) {
	response := SuccessResponse{
		Data:      data,
		Success:   true,
		Timestamp: time.Now().UTC(),
	}

	if len(message) > 0 {
		response.Message = message[0]
	}

	if requestID := c.GetString("request_id"); requestID != "" {
		response.RequestID = requestID
	}

	c.JSON(http.StatusOK, response)
}

// RespondCreated sends a 201 Created response
func RespondCreated(c *gin.Context, data interface{}, message ...string) {
	response := SuccessResponse{
		Data:      data,
		Success:   true,
		Timestamp: time.Now().UTC(),
	}

	if len(message) > 0 {
		response.Message = message[0]
	}

	if requestID := c.GetString("request_id"); requestID != "" {
		response.RequestID = requestID
	}

	c.JSON(http.StatusCreated, response)
}

// RespondNoContent sends a 204 No Content response
func RespondNoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Common validation error messages
var (
	ErrInvalidEmail       = NewValidationError("Invalid email format")
	ErrPasswordTooShort   = NewValidationError("Password must be at least 8 characters")
	ErrRequiredField      = NewValidationError("Required field is missing")
	ErrInvalidImageFormat = NewValidationError("Invalid image format. Only JPEG and PNG are supported")
	ErrFileTooLarge       = NewValidationError("File size exceeds maximum allowed size")
	ErrInvalidRequestBody = NewValidationError("Invalid request body format")
)

// Common authentication error messages
var (
	ErrUnauthorized     = NewAuthError("Authentication required")
	ErrInvalidToken     = NewAuthError("Invalid or expired token")
	ErrTokenMissing     = NewAuthError("Authorization token is missing")
	ErrInvalidCredentials = NewAuthError("Invalid credentials")
)

// Common permission error messages
var (
	ErrForbidden          = NewPermissionError("Access denied")
	ErrInsufficientRole   = NewPermissionError("Insufficient role permissions")
	ErrResourceAccess     = NewPermissionError("You don't have access to this resource")
)

// Common resource error messages
var (
	ErrImageNotFound = NewNotFoundError("Image")
	ErrUserNotFound  = NewNotFoundError("User")
)

// Common business logic error messages
var (
	ErrQuotaExceeded     = NewQuotaExceededError("Storage quota exceeded")
	ErrDuplicateEmail    = NewConflictError("Email address already exists")
	ErrImageLimit        = NewConflictError("Maximum number of images reached")
	ErrProcessingFailed  = NewInternalError("Image processing failed")
)
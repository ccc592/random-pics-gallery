package handlers

import (
	"strings"
	"time"
)

// ValidationError represents a field validation error
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// ErrorResponse represents the standard error response structure
type ErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Details string `json:"details,omitempty"`
	} `json:"error"`
	Validations []ValidationError `json:"validations,omitempty"`
	RequestID   string            `json:"request_id,omitempty"`
	Timestamp   string            `json:"timestamp"`
}

// CreateErrorResponse creates a standardized error response
func CreateErrorResponse(code int, message, details string) ErrorResponse {
	return ErrorResponse{
		Error: struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Details string `json:"details,omitempty"`
		}{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// CreateValidationErrorResponse creates an error response with validation errors
func CreateValidationErrorResponse(message string, validations []ValidationError) ErrorResponse {
	return ErrorResponse{
		Error: struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Details string `json:"details,omitempty"`
		}{
			Code:    400, // Bad Request
			Message: message,
		},
		Validations: validations,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
}

// ParseTagsFromString parses a comma-separated tags string into a slice
func ParseTagsFromString(tagsPtr *string) []string {
	if tagsPtr == nil || *tagsPtr == "" {
		return []string{}
	}

	tagList := strings.Split(*tagsPtr, ",")
	tags := make([]string, len(tagList))
	for i, tag := range tagList {
		tags[i] = strings.TrimSpace(tag)
	}

	return tags
}
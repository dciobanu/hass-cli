// Package api provides HTTP client functionality for Home Assistant API.
package api

import (
	"errors"
	"fmt"
)

// APIError represents an error from the Home Assistant API.
type APIError struct {
	StatusCode int
	Message    string
	Code       string // e.g., "invalid_format", "unauthorized"
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s (%s, HTTP %d)", e.Message, e.Code, e.StatusCode)
	}
	return fmt.Sprintf("%s (HTTP %d)", e.Message, e.StatusCode)
}

// Common API errors
var (
	ErrUnauthorized = &APIError{
		StatusCode: 401,
		Message:    "Invalid or missing access token",
		Code:       "unauthorized",
	}

	ErrNotFound = &APIError{
		StatusCode: 404,
		Message:    "Resource not found",
		Code:       "not_found",
	}

	ErrBadRequest = &APIError{
		StatusCode: 400,
		Message:    "Bad request",
		Code:       "bad_request",
	}

	ErrServerError = &APIError{
		StatusCode: 500,
		Message:    "Internal server error",
		Code:       "server_error",
	}
)

// IsUnauthorized returns true if the error is an authorization error.
func IsUnauthorized(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 401
	}
	return false
}

// IsNotFound returns true if the error is a not found error.
func IsNotFound(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 404
	}
	return false
}

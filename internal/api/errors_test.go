package api

import (
	"errors"
	"fmt"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *APIError
		want string
	}{
		{
			name: "with code",
			err:  &APIError{StatusCode: 401, Message: "Unauthorized", Code: "unauthorized"},
			want: "Unauthorized (unauthorized, HTTP 401)",
		},
		{
			name: "without code",
			err:  &APIError{StatusCode: 500, Message: "Server error"},
			want: "Server error (HTTP 500)",
		},
		{
			name: "predefined unauthorized",
			err:  ErrUnauthorized,
			want: "Invalid or missing access token (unauthorized, HTTP 401)",
		},
		{
			name: "predefined not found",
			err:  ErrNotFound,
			want: "Resource not found (not_found, HTTP 404)",
		},
		{
			name: "predefined bad request",
			err:  ErrBadRequest,
			want: "Bad request (bad_request, HTTP 400)",
		},
		{
			name: "predefined server error",
			err:  ErrServerError,
			want: "Internal server error (server_error, HTTP 500)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsUnauthorized(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "APIError 401",
			err:  &APIError{StatusCode: 401, Message: "test"},
			want: true,
		},
		{
			name: "predefined ErrUnauthorized",
			err:  ErrUnauthorized,
			want: true,
		},
		{
			name: "APIError 404",
			err:  &APIError{StatusCode: 404, Message: "test"},
			want: false,
		},
		{
			name: "wrapped APIError 401",
			err:  fmt.Errorf("wrapped: %w", &APIError{StatusCode: 401, Message: "test"}),
			want: true,
		},
		{
			name: "non-APIError",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsUnauthorized(tt.err)
			if got != tt.want {
				t.Errorf("IsUnauthorized() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "APIError 404",
			err:  &APIError{StatusCode: 404, Message: "test"},
			want: true,
		},
		{
			name: "predefined ErrNotFound",
			err:  ErrNotFound,
			want: true,
		},
		{
			name: "APIError 401",
			err:  &APIError{StatusCode: 401, Message: "test"},
			want: false,
		},
		{
			name: "wrapped APIError 404",
			err:  fmt.Errorf("wrapped: %w", &APIError{StatusCode: 404, Message: "test"}),
			want: true,
		},
		{
			name: "non-APIError",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotFound(tt.err)
			if got != tt.want {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

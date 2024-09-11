package apimodels

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// apierror represents a standardized error response for the api.
//
// it encapsulates:
//   - an http status code
//   - a human-readable error message
//
// purpose:
//   - primarily used to send synchronous errors from the orchestrator endpoint
//   - provides a structured json error response to http clients
//
// usage:
//   - when the cli interacts with an orchestrator node via an http client:
//     1. the http client receives this structured json error
//     2. the human-readable message is output to stdout
//     3. the http status code allows for programmatic handling of different error types
//
// benefits:
//   - ensures consistent error formatting across the api
//   - facilitates both user-friendly error messages and machine-readable error codes
type APIError struct {
	// httpstatuscode is the http status code associated with this error.
	// it should correspond to standard http status codes (e.g., 400, 404, 500).
	HTTPStatusCode int `json:"code"`

	// message is a short, human-readable description of the error.
	// it should be concise and provide a clear indication of what went wrong.
	Message string `json:"message"`

	// RequestID is the request ID of the request that caused the error.
	RequestID string `json:"request_id"`
}

// NewAPIError creates a new APIError with the given HTTP status code and message.
func NewAPIError(statusCode int, message string) *APIError {
	return &APIError{
		HTTPStatusCode: statusCode,
		Message:        message,
	}
}

// NewBadRequestError creates an APIError for Bad Request (400) errors.
func NewBadRequestError(message string) *APIError {
	return NewAPIError(http.StatusBadRequest, message)
}

// NewUnauthorizedError creates an APIError for Unauthorized (401) errors.
func NewUnauthorizedError(message string) *APIError {
	return NewAPIError(http.StatusUnauthorized, message)
}

// NewForbiddenError creates an APIError for Forbidden (403) errors.
func NewForbiddenError(message string) *APIError {
	return NewAPIError(http.StatusForbidden, message)
}

// NewNotFoundError creates an APIError for Not Found (404) errors.
func NewNotFoundError(message string) *APIError {
	return NewAPIError(http.StatusNotFound, message)
}

// NewConflictError creates an APIError for Conflict (409) errors.
func NewConflictError(message string) *APIError {
	return NewAPIError(http.StatusConflict, message)
}

// NewInternalServerError creates an APIError for Internal Server Error (500) errors.
func NewInternalServerError(message string) *APIError {
	return NewAPIError(http.StatusInternalServerError, message)
}

func NewJobNotFound(jobID string) *APIError {
	return NewAPIError(http.StatusNotFound, fmt.Sprintf("job id %s not found", jobID))
}

// IsNotFound checks if the error is an APIError with a Not Found status.
func IsNotFound(err error) bool {
	apiErr, ok := err.(*APIError)
	return ok && apiErr.HTTPStatusCode == http.StatusNotFound
}

// IsBadRequest checks if the error is an APIError with a Bad Request status.
func IsBadRequest(err error) bool {
	apiErr, ok := err.(*APIError)
	return ok && apiErr.HTTPStatusCode == http.StatusBadRequest
}

// IsInternalServerError checks if the error is an APIError with an Internal Server Error status.
func IsInternalServerError(err error) bool {
	apiErr, ok := err.(*APIError)
	return ok && apiErr.HTTPStatusCode == http.StatusInternalServerError
}

// Error implements the error interface, allowing APIError to be used as a standard Go error.
func (e *APIError) Error() string {
	return e.Message
}

// Parse HTTP Resposne to APIError
func FromHttpResponse(resp *http.Response) (*APIError, error) {

	if resp == nil {
		return nil, errors.New("response is nil, cannot be unmarsheld to APIError")
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var apiErr APIError
	err = json.Unmarshal(body, &apiErr)
	if err != nil {
		return nil, fmt.Errorf("error parsing response body: %w", err)
	}

	// If the JSON didn't include a status code, use the HTTP Status
	if apiErr.HTTPStatusCode == 0 {
		apiErr.HTTPStatusCode = resp.StatusCode
	}

	return &apiErr, nil
}

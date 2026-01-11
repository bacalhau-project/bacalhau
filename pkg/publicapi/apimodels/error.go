package apimodels

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
)

// APIError represents a standardized error response for the api.
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
	HTTPStatusCode int `json:"Status"`

	// message is a short, human-readable description of the error.
	// it should be concise and provide a clear indication of what went wrong.
	Message string `json:"Message"`

	// RequestID is the request ID of the request that caused the error.
	RequestID string `json:"RequestID"`

	// Code is the error code of the error.
	Code string `json:"Code"`

	// Component is the component that caused the error.
	Component string `json:"Component"`

	// Hint is a string providing additional context or suggestions related to the error.
	Hint string `json:"Hint,omitempty"`

	// Details is a map of string key-value pairs providing additional error context.
	Details map[string]string `json:"Details,omitempty"`
}

// NewAPIError creates a new APIError with the given HTTP status code and message.
func NewAPIError(statusCode int, message string) *APIError {
	return &APIError{
		HTTPStatusCode: statusCode,
		Message:        message,
		Details:        make(map[string]string),
	}
}

// NewUnauthorizedError creates an APIError for Unauthorized (401) errors.
func NewUnauthorizedError(message string) *APIError {
	return NewAPIError(http.StatusUnauthorized, message)
}

// Error implements the error interface, allowing APIError to be used as a standard Go error.
func (e *APIError) Error() string {
	return e.Message
}

// Parse HTTP Response to APIError
func GenerateAPIErrorFromHTTPResponse(resp *http.Response) *APIError {
	if resp == nil {
		return NewAPIError(0, "API call error, invalid response")
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return NewAPIError(
			resp.StatusCode,
			fmt.Sprintf("Unable to read API call response body. Error: %q", err.Error()))
	}

	var apiErr APIError
	err = json.Unmarshal(body, &apiErr)
	if err != nil {
		return NewAPIError(
			resp.StatusCode,
			fmt.Sprintf("Unable to parse API call response body. Error: %q. Body received: %q",
				err.Error(),
				string(body),
			))
	}

	// If the JSON didn't include a status code, use the HTTP Status
	if apiErr.HTTPStatusCode == 0 {
		apiErr.HTTPStatusCode = resp.StatusCode
	}

	return &apiErr
}

// FromBacError converts a bacerror.Error to an APIError
func FromBacError(err bacerrors.Error) *APIError {
	return &APIError{
		HTTPStatusCode: err.HTTPStatusCode(),
		Message:        err.Error(),
		Code:           string(err.Code()),
		Component:      err.Component(),
		Hint:           err.Hint(),
		Details:        err.Details(),
	}
}

// ToBacError converts an APIError to a bacerror.Error
func (e *APIError) ToBacError() bacerrors.Error {
	details := e.Details
	if details == nil {
		details = make(map[string]string)
	}
	details["request_id"] = e.RequestID
	return bacerrors.New(e.Error()).
		WithHTTPStatusCode(e.HTTPStatusCode).
		WithCode(bacerrors.Code(e.Code)).
		WithComponent(e.Component).
		WithHint(e.Hint).
		WithDetails(details)
}

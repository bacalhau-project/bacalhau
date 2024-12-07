package ncl

import (
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
)

const (
	// ErrorMessageType is used when responding with an error
	ErrorMessageType = "NCL-Error"

	// KeyStatusCode is the key for the status code
	KeyStatusCode = "Bacalhau-StatusCode"

	// StatusBadRequest is the status code for a bad request
	StatusBadRequest = 400

	// StatusNotFound is the status code for a not handler found
	StatusNotFound = 404

	// StatusServerError is the status code for a server error
	StatusServerError = 500
)

// ErrorResponse is used to respond with an error
type ErrorResponse struct {
	StatusCode int    `json:"StatusCode"`
	Message    string `json:"Message"`
}

// NewErrorResponse creates a new error response
func NewErrorResponse(statusCode int, message string) ErrorResponse {
	return ErrorResponse{
		StatusCode: statusCode,
		Message:    message,
	}
}

// Error returns the error message
func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("status code: %d, message: %s", e.StatusCode, e.Message)
}

// ToEnvelope converts the error to an envelope
func (e *ErrorResponse) ToEnvelope() *envelope.Message {
	errMsg := envelope.NewMessage(e)
	errMsg.WithMetadataValue(envelope.KeyMessageType, ErrorMessageType)
	errMsg.WithMetadataValue(KeyStatusCode, fmt.Sprintf("%d", e.StatusCode))
	errMsg.WithMetadataValue(KeyEventTime, time.Now().Format(time.RFC3339))
	return errMsg
}

// compile-time check for interface conformance
var _ error = &ErrorResponse{}

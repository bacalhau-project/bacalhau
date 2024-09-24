package models

import (
	"errors"
	"fmt"
	"net/http"
)

type ErrorCode string

const (
	DetailsKeyIsError        = "IsError"
	DetailsKeyHint           = "Hint"
	DetailsKeyRetryable      = "Retryable"
	DetailsKeyFailsExecution = "FailsExecution"
	DetailsKeyNewState       = "NewState"
)

const (
	BadRequestError    ErrorCode = "BadRequest"
	InternalError      ErrorCode = "InternalError"
	NotFoundError      ErrorCode = "NotFound"
	ServiceUnavailable ErrorCode = "ServiceUnavailable"
	NotImplemented     ErrorCode = "NotImplemented"
	ResourceExhausted  ErrorCode = "ResourceExhausted"
	ResourceInUse      ErrorCode = "ResourceInUse"
	VersionMismatch    ErrorCode = "VersionMismatch"
	ValidationFailed   ErrorCode = "ValidationFailed"
	TooManyRequests    ErrorCode = "TooManyRequests"
	NetworkFailure     ErrorCode = "NetworkFailure"
	ConfigurationError ErrorCode = "ConfigurationError"
	DatastoreFailure   ErrorCode = "DatastoreFailure"
)

type HasHint interface {
	// Hint A human-readable string that advises the user on how they might solve the error.
	Hint() string
}

type HasRetryable interface {
	// Retryable Whether the error could be retried, assuming the same input and
	// node configuration; i.e. the error is transient and due to network
	// capacity or service outage.
	//
	// If a component raises an error with Retryable() as true, the system may
	// retry the last action after some length of time. If it is false, it
	// should not try the action again, and may choose an alternative action or
	// fail the job.
	Retryable() bool
}

type HasFailsExecution interface {
	// FailsExecution Whether this error means that the associated execution cannot
	// continue.
	//
	// If a component raises an error with FailsExecution() as true,
	// the hosting executor should report the execution as failed and stop any
	// further steps.
	FailsExecution() bool
}

type HasDetails interface {
	// Details An extra set of metadata provided by the error.
	Details() map[string]string
}

type HasCode interface {
	// Details a code
	Code() ErrorCode
}

// HasHTTPStatusCode is an interface that defines a method for retrieving
// an HTTP status code associated with an error.
type HasHTTPStatusCode interface {
	// HTTPStatusCode returns the HTTP status code associated with the error.
	// This can be useful for mapping internal errors to appropriate HTTP responses.
	HTTPStatusCode() int
}

// BaseError is a custom error type in Go that provides additional fields
// and methods for more detailed error handling. It implements the error
// interface, as well as additional interfaces for providing a hint,
// indicating whether the error is retryable, whether it fails execution,
// and for providing additional details.
type BaseError struct {
	message        string
	hint           string
	retryable      bool
	failsExecution bool
	component      string
	httpStatusCode int
	details        map[string]string
	code           ErrorCode
}

// IsBaseError is a helper function that checks if an error is a BaseError.
func IsBaseError(err error) bool {
	var baseError *BaseError
	ok := errors.As(err, &baseError)
	return ok
}

// NewBaseError is a constructor function that creates a new BaseError with
// only the message field set.
func NewBaseError(format string, a ...any) *BaseError {
	return &BaseError{
		httpStatusCode: 0,
		component:      "Bacalhau",
		message:        fmt.Sprintf(format, a...),
	}
}

// WithHint is a method that sets the hint field of BaseError and returns
// the BaseError itself for chaining.
func (e *BaseError) WithHint(hint string) *BaseError {
	e.hint = hint
	return e
}

// WithRetryable is a method that sets the retryable field of BaseError and
// returns the BaseError itself for chaining.
func (e *BaseError) WithRetryable() *BaseError {
	e.retryable = true
	return e
}

// WithFailsExecution is a method that sets the failsExecution field of
// BaseError and returns the BaseError itself for chaining.
func (e *BaseError) WithFailsExecution() *BaseError {
	e.failsExecution = true
	return e
}

// WithDetails is a method that sets the details field of BaseError and
// returns the BaseError itself for chaining.
func (e *BaseError) WithDetails(details map[string]string) *BaseError {
	e.details = details
	return e
}

// WithCode is a method that sets the code field of BaseError and
// returns the BaseError itself for chaining
func (e *BaseError) WithCode(code ErrorCode) *BaseError {
	e.code = code
	return e
}

// WithHTTPStatusCode is a method that sets the httpStatusCode field of BaseError and
// returns the BaseError itself for chaining. This method allows setting a specific
// HTTP status code associated with the error, which can be useful when translating
// the error into an HTTP response.
func (e *BaseError) WithHTTPStatusCode(statusCode int) *BaseError {
	e.httpStatusCode = statusCode
	return e
}

// WithComponent is a method that sets the component field of BaseError and
// returns the BaseError itself for chaining. This method allows specifying
// which component of the system generated the error, providing more context
// for debugging and error handling.
func (e *BaseError) WithComponent(component string) *BaseError {
	e.component = component
	return e
}

// Error is a method that returns the message field of BaseError. This
// method makes BaseError satisfy the error interface.
func (e *BaseError) Error() string {
	return e.message
}

// Hint is a method that returns the hint field of BaseError.
func (e *BaseError) Hint() string {
	return e.hint
}

// Retryable is a method that returns the retryable field of BaseError.
func (e *BaseError) Retryable() bool {
	return e.retryable
}

// FailsExecution is a method that returns the failsExecution field of BaseError.
func (e *BaseError) FailsExecution() bool {
	return e.failsExecution
}

// Details is a method that returns the details field of BaseError.
func (e *BaseError) Details() map[string]string {
	return e.details
}

// Code returns a unique code to identify the error
func (e *BaseError) Code() ErrorCode {
	return e.code
}

// Component is a method that returns the component field of BaseError.
func (e *BaseError) Component() string {
	return e.component
}

// HTTPStatusCode is a method that returns the httpStatusCode field of BaseError.
// If no specific HTTP status code has been set, it returns 0.
// This method can be used to retrieve the HTTP status code associated with the error,
// which is useful when translating the error into an HTTP response.
func (e *BaseError) HTTPStatusCode() int {
	if e.httpStatusCode != 0 {
		return e.httpStatusCode
	}
	return inferHTTPStatusCode(e.code)
}

func inferHTTPStatusCode(code ErrorCode) int {
	switch code {
	case BadRequestError, ValidationFailed:
		return http.StatusBadRequest
	case NotFoundError:
		return http.StatusNotFound
	case ServiceUnavailable:
		return http.StatusServiceUnavailable
	case NotImplemented:
		return http.StatusNotImplemented
	case ResourceExhausted:
		return http.StatusTooManyRequests
	case ResourceInUse:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func IsErrorWithCode(err error, code ErrorCode) bool {
	var baseErr *BaseError
	if errors.As(err, &baseErr) {
		errCode := baseErr.Code()
		if errCode == code {
			return true
		}
	}
	return false
}

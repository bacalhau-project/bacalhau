// Package bacerrors provides a rich error type for detailed error handling in Go applications.
// It offers functionality for error wrapping, stack trace tracking, and additional context
// such as hints, retryability, and HTTP status codes.
package bacerrors

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

// Error interface defines methods that provide additional fields
// and functionality for more detailed error handling. It implements the standard
// error interface and adds methods for providing hints, indicating retryability,
// execution impact, and additional details.
type Error interface {
	error

	// Hint returns a string providing additional context or suggestions related to the error.
	Hint() string

	// Retryable indicates whether the operation that caused this error can be retried.
	Retryable() bool

	// FailsExecution indicates whether this error should cause the overall execution to fail.
	FailsExecution() bool

	// Details returns a map of string key-value pairs providing additional error context.
	Details() map[string]string

	// Code returns the ErrorCode associated with this error.
	Code() ErrorCode

	// Component returns the name of the system component where the error originated.
	Component() string

	// HTTPStatusCode returns the HTTP status code associated with this error.
	HTTPStatusCode() int

	// ErrorWrapped returns the full error message, including messages from wrapped errors.
	ErrorWrapped() string

	// Unwrap returns the next error in the error chain, if any.
	Unwrap() error

	// StackTrace returns a string representation of the stack trace where the error occurred.
	StackTrace() string

	// WithHint adds or updates the hint associated with the error.
	WithHint(hintFormat string, a ...any) Error

	// WithRetryable marks the error as retryable.
	WithRetryable() Error

	// WithFailsExecution marks the error as causing execution failure.
	WithFailsExecution() Error

	// WithDetails adds or updates the details associated with the error.
	WithDetails(details map[string]string) Error

	// WithDetail adds or updates a single detail associated with the error.
	WithDetail(key, value string) Error

	// WithCode sets the ErrorCode for this error.
	WithCode(code ErrorCode) Error

	// WithHTTPStatusCode sets the HTTP status code associated with this error.
	WithHTTPStatusCode(statusCode int) Error

	// WithComponent sets the component name where the error originated.
	WithComponent(component string) Error
}

// errorImpl is a concrete implementation of the Error interface
type errorImpl struct {
	cause          string
	hint           string
	retryable      bool
	failsExecution bool
	component      string
	httpStatusCode int
	details        map[string]string
	code           ErrorCode
	wrappedErr     error
	wrappingMsg    string
	stack          *stack
}

// Ensure errorImpl implements Error interface
var _ Error = (*errorImpl)(nil)

// New creates a new Error with only the message field set.
// It initializes the error with a stack trace and sets the component to "Bacalhau" by default.
func New(format string, a ...any) Error {
	message := fmt.Sprintf(format, a...)
	return &errorImpl{
		cause:     message,
		component: "Bacalhau",
		stack:     callers(),
	}
}

// Wrap creates a new Error that wraps an existing error.
// If the wrapped error is already a bacerrors.Error, it updates the wrapped error
// while preserving the original error's information. Otherwise, it creates a new Error
// that includes both the new message and the original error's message.
func Wrap(err error, format string, a ...any) Error {
	message := fmt.Sprintf(format, a...)
	var bacErr *errorImpl
	if errors.As(err, &bacErr) {
		// If it's already a bacerror, just update the wrapped error
		newErr := *bacErr // Create a copy
		// Update the wrapped error
		newErr.wrappedErr = err
		newErr.wrappingMsg = message
		return &newErr
	}
	nErr := New("%s: %s", message, err.Error())
	nErr.(*errorImpl).wrappedErr = err
	nErr.(*errorImpl).wrappingMsg = message
	return nErr
}

// WithHint sets the hint field of Error and returns
// the Error itself for chaining.
func (e *errorImpl) WithHint(hintFormat string, a ...any) Error {
	e.hint = fmt.Sprintf(hintFormat, a...)
	return e
}

// WithRetryable sets the retryable field of Error and
// returns the Error itself for chaining.
func (e *errorImpl) WithRetryable() Error {
	e.retryable = true
	return e
}

// WithFailsExecution sets the failsExecution field of
// Error and returns the Error itself for chaining.
func (e *errorImpl) WithFailsExecution() Error {
	e.failsExecution = true
	return e
}

// WithDetails sets the details field of Error and
// returns the Error itself for chaining.
func (e *errorImpl) WithDetails(details map[string]string) Error {
	// merge the new details with the existing details
	if e.details == nil {
		e.details = make(map[string]string)
	}
	for k, v := range details {
		e.details[k] = v
	}
	return e
}

// WithDetail adds a single detail to the details field of Error and
// returns the Error itself for chaining.
func (e *errorImpl) WithDetail(key, value string) Error {
	if e.details == nil {
		e.details = make(map[string]string)
	}
	e.details[key] = value
	return e
}

// WithCode sets the code field of Error and
// returns the Error itself for chaining
func (e *errorImpl) WithCode(code ErrorCode) Error {
	e.code = code
	return e
}

// WithHTTPStatusCode sets the httpStatusCode field of Error and
// returns the Error itself for chaining. This method allows setting a specific
// HTTP status code associated with the error, which can be useful when translating
// the error into an HTTP response.
func (e *errorImpl) WithHTTPStatusCode(statusCode int) Error {
	e.httpStatusCode = statusCode
	return e
}

// WithComponent sets the component field of Error and
// returns the Error itself for chaining. This method allows specifying
// which component of the system generated the error, providing more context
// for debugging and error handling.
func (e *errorImpl) WithComponent(component string) Error {
	e.component = component
	return e
}

// Error returns the message field of Error. This
// method makes Error satisfy the error interface.
func (e *errorImpl) Error() string {
	return e.cause
}

// ErrorWrapped returns the full error message, including all wrapped error messages.
// This method is useful when an error is wrapped by another error, and the complete
// error chain is needed.
func (e *errorImpl) ErrorWrapped() string {
	if e.wrappedErr != nil {
		// if the wrapped error is also a bacerror, return the wrapping message
		var wErr *errorImpl
		if errors.As(e.wrappedErr, &wErr) {
			return fmt.Sprintf("%s: %s", e.wrappingMsg, wErr.ErrorWrapped())
		}
		return fmt.Sprintf("%s: %s", e.wrappingMsg, e.wrappedErr)
	}
	return e.Error()
}

// Unwrap returns the wrapped error, if any.
// This method supports Go 1.13+ error unwrapping.
func (e *errorImpl) Unwrap() error {
	return e.wrappedErr
}

// StackTrace returns a string representation of the stack trace captured
// when the error was created or wrapped.
func (e *errorImpl) StackTrace() string {
	return e.stack.String()
}

// Hint returns the hint field of Error.
func (e *errorImpl) Hint() string {
	return e.hint
}

// Retryable returns the retryable field of Error.
func (e *errorImpl) Retryable() bool {
	return e.retryable
}

// FailsExecution returns the failsExecution field of Error.
func (e *errorImpl) FailsExecution() bool {
	return e.failsExecution
}

// Details returns the details field of Error.
func (e *errorImpl) Details() map[string]string {
	return e.details
}

// Code returns a unique code to identify the error
func (e *errorImpl) Code() ErrorCode {
	return e.code
}

// Component returns the component field of Error.
func (e *errorImpl) Component() string {
	return e.component
}

// HTTPStatusCode returns the httpStatusCode field of Error.
// If no specific HTTP status code has been set, it returns the result of inferHTTPStatusCode.
// This method can be used to retrieve the HTTP status code associated with the error,
// which is useful when translating the error into an HTTP response.
func (e *errorImpl) HTTPStatusCode() int {
	if e.httpStatusCode != 0 {
		return e.httpStatusCode
	}
	return inferHTTPStatusCode(e.code)
}

// inferHTTPStatusCode maps ErrorCode to appropriate HTTP status codes.
// This function is used when a specific HTTP status code hasn't been set for an error.
func inferHTTPStatusCode(code ErrorCode) int {
	switch code {
	case BadRequestError, ValidationError:
		return http.StatusBadRequest
	case NotFoundError:
		return http.StatusNotFound
	case UnauthorizedError:
		return http.StatusUnauthorized
	case Forbidden:
		return http.StatusForbidden
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

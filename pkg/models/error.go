package models

import "fmt"

const (
	DetailsKeyIsError        = "IsError"
	DetailsKeyHint           = "Hint"
	DetailsKeyRetryable      = "Retryable"
	DetailsKeyFailsExecution = "FailsExecution"
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
	Code() int
}

type ErrorCode struct {
	component      string
	httpStatusCode int
}

func NewErrorCode(component string, httpStatusCode int) ErrorCode {
	return ErrorCode{component: component, httpStatusCode: httpStatusCode}
}

func (ec ErrorCode) HTTPStatusCode() int {
	if ec.httpStatusCode == 0 {
		return 500
	}
	return ec.httpStatusCode
}

func (ec ErrorCode) Component() string {
	return ec.component
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
	details        map[string]string
	code           ErrorCode
}

// NewBaseError is a constructor function that creates a new BaseError with
// only the message field set.
func NewBaseError(format string, a ...any) *BaseError {
	return &BaseError{message: fmt.Sprintf(format, a...)}
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

// Details a Unique Code to identify the error
func (e *BaseError) Code() ErrorCode {
	return e.code
}

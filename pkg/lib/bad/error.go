// Package bad helps you deal with errors.
package bad

import (
	"strings"
)

// ErrorSubject identifies where the error arose according to the component that
// emitted it.
type ErrorSubject string

const (
	// The error arose for an unspecified reason.
	ErrSubjectNone ErrorSubject = ""
	// The error arose because user input did not conform to the required
	// specification. The user will need to modify their request before
	// submitting it again.
	ErrSubjectInput ErrorSubject = "input"
	// The error arose because of an internal logic problem with the component
	// that created the error, normally signaling a programming issue. The
	// system developer will need to modify their code to meet this request.
	ErrSubjectInternal ErrorSubject = "internal"
	// The error arose because some other service or component that the
	// component that created the error depends on also returned an error. The
	// error may be transient, especially if this error is a leaf, or may
	// require operator intervention (e.g. out of disk space).
	ErrSubjectDependency ErrorSubject = "dependency"
)

// Error represents a node in a tree of errors. Error (and thus the whole tree)
// is JSON marshallable so that a tree of Errors can be sent across HTTP.
type Error struct {
	// The type of the error. For native errors, this is just the error text,
	// but more structured errors this is a unique identifier that clients can
	// use to display certain types of important errors in specific ways.
	// Essentially, the "type" identifies the type of thing in the "data" field,
	// and allows the client to unmarshall the data field appropriately.
	//
	// For example, if a client knows how to represent errors about resource
	// limits being unmet, the type may be "Unmet resource limits" and the data
	// field may include structured information about what resource limits are
	// being unmet that the client can choose to display in a number of ways.
	Type string `json:"type"`
	Data any    `json:"data"`

	// The subject of the error, which identifies where the problem occurred and
	// is used to select appropriate status codes and retry information.
	//
	// Not every Error needs to have a Subject specified – instead subjects can
	// be added to tag any error from a certain place. For example, a Validate()
	// method will almost always be returning an error due to user input, so any
	// error emitted from this place can be wrapped with a ErrSubjectInput.
	Subject ErrorSubject `json:"subject"`

	// Any child errors that this error wraps.
	Errs []Error `json:"errs"`
}

// Error() is necessary for Error to implement the error interface, which makes
// it much easier to pass back from functions. We shouldn't actually use this
// except in internal logging, but it implements a simple printer that will
// output the entire tree of errors, skipping any blank levels.
func (err *Error) Error() string {
	var out strings.Builder
	if err == nil {
		return ""
	}

	indent := ""
	if len(err.Type) > 0 {
		out.WriteString("* ")
		out.WriteString(err.Type)
		indent = "\t"
		if len(err.Errs) > 0 {
			out.WriteString(": ")
		}
		out.WriteRune('\n')
	}
	for _, subError := range err.Errs {
		for _, line := range strings.SplitAfter(subError.Error(), "\n") {
			if line == "" {
				continue
			}
			out.WriteString(indent)
			out.WriteString(line)
		}
	}
	return out.String()
}

var _ error = (*Error)(nil)

// Leaves returns just the Errors from the tree that do not have any children.
// These Errors are normally the ones with the most specific error information.
// The order in which the leaves are returned is unspecified.
func (err *Error) Leaves() []Error {
	if err == nil {
		return nil
	}

	if len(err.Errs) == 0 {
		return []Error{*err}
	}

	leaves := make([]Error, 0, len(err.Errs))
	for _, subErr := range err.Errs {
		leaves = append(leaves, subErr.Leaves()...)
	}
	return leaves
}

// Input transforms the error to report that it was the result of user input not
// meeting requirements. Data is an optional parameter that should be specified
// 0 or 1 times and will replace any data in the error if specified.
func Input(err error, data ...any) error {
	e := ToError(err)
	if e == nil {
		return nil
	}
	e.Subject = ErrSubjectInput
	if len(data) > 1 {
		panic("too much error data passed to Input()")
	}
	if len(data) > 0 {
		e.Data = data[0]
	}
	return e
}

type singleWrapper interface {
	error
	Unwrap() error
}

type multiWrapper interface {
	error
	Unwrap() []error
}

// ToError converts a Go native error in an Error. If the error is already an
// Error, it is returned unchanged. If the error is nil, nil is returned. If the
// error wraps other errors, they are also converted to Errors and the full tree
// of errors returned.
func ToError(err error) *Error {
	if err == nil {
		return nil
	}

	if apiError, ok := (err).(*Error); ok {
		return apiError
	}

	// If this error could wrap another error, make an error and add its single
	// wrapped error as an Errs.
	var subErrs []error
	if singleErr, ok := (err).(singleWrapper); ok {
		subErrs = append(subErrs, singleErr.Unwrap())
	} else if multiErr, ok := (err).(multiWrapper); ok {
		subErrs = multiErr.Unwrap()
	}

	apiError := &Error{
		Type: err.Error(),
		Errs: make([]Error, 0, len(subErrs)),
	}

	// Discard this error.Error() if its repeating the wrapped error messages,
	// but obviously not if it is a leaf error with no sub-errors. The logic
	// here is that calling Error() on errors that wrap other errors often just
	// returns the concatenation of the wrapped Error() calls, which is not
	// useful to have in the Type field.
	for _, subErr := range subErrs {
		apiError.Type = strings.Replace(apiError.Type, subErr.Error(), "", 1)

		subAPIErr := ToError(subErr)
		if subAPIErr != nil {
			apiError.Errs = append(apiError.Errs, *subAPIErr)
		}
	}
	apiError.Type = strings.TrimRight(strings.TrimSpace(apiError.Type), ":")

	return apiError
}

package bacerrors

import (
	"errors"
)

// IsError is a helper function that checks if an error is an Error.
func IsError(err error) bool {
	return FromError(err) != nil
}

// IsErrorWithCode checks if an error is an Error with a specific ErrorCode.
func IsErrorWithCode(err error, code ErrorCode) bool {
	var customErr Error
	if errors.As(err, &customErr) {
		return customErr.Code() == code
	}
	return false
}

// FromError converts an error to an Error.
func FromError(err error) Error {
	var customErr Error
	if errors.As(err, &customErr) {
		return customErr
	}
	return nil
}

package bacerrors

import (
	"errors"
)

// IsError is a helper function that checks if an error is an Error.
func IsError(err error) bool {
	var customErr Error
	ok := errors.As(err, &customErr)
	return ok
}

// IsErrorWithCode checks if an error is an Error with a specific ErrorCode.
func IsErrorWithCode(err error, code ErrorCode) bool {
	var customErr Error
	if errors.As(err, &customErr) {
		return customErr.Code() == code
	}
	return false
}

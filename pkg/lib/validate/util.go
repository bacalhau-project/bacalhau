package validate

import (
	"errors"
	"fmt"
)

// createError constructs an error with a message.
// 'msg' is the message to be used for the error.
// It returns an error with the provided message.
func createError(msg string) error {
	if msg == "" {
		// If the message is empty, return a generic error.
		return errors.New("an error occurred")
	}
	// Create and return the error with the provided message.
	return errors.New(msg)
}

// createErrorf constructs an error with a formatted message.
// 'msg' is a format string and 'args' are the values to be formatted into the message.
func createErrorf(msg string, args ...any) error {
	if len(args) == 0 {
		// If no arguments, return the message as-is.
		return errors.New(msg)
	}
	// If arguments are provided, format the message.
	return fmt.Errorf(msg, args...)
}

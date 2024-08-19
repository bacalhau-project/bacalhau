package validate

import "fmt"

// createError constructs an error with a formatted message.
// 'msg' is a format string and 'args' are the values to be formatted into the message.
func createError(msg string, args ...any) error {
	if len(args) == 0 {
		// If no arguments, return the message as-is.
		return fmt.Errorf("%s", msg)
	}
	// If arguments are provided, format the message.
	return fmt.Errorf(msg, args...)
}

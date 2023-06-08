package handler

const (
	// ExitSuccess Successful termination
	ExitSuccess = 0

	// ExitError Catchall for general errors
	ExitError = 1

	// ExitBuiltinMisuse Misuse of shell builtins (according to Bash documentation)
	ExitBuiltinMisuse = 2

	// ExitCannotExecute Command invoked cannot execute
	ExitCannotExecute = 126

	// ExitCommandNotFound Command not found
	ExitCommandNotFound = 127

	// ExitOutOfRange Exit status out of range
	ExitOutOfRange = 255
)

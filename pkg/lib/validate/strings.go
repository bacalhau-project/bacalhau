package validate

import (
	"strings"
	"unicode"
)

// NotBlank checks if the provided string is not empty or consisting only of whitespace.
// It returns an error if the string is blank, using the provided message and arguments.
func NotBlank(s string, msg string, args ...any) error {
	if strings.TrimSpace(s) == "" {
		return createError(msg, args...)
	}
	return nil
}

// NoSpaces checks if the provided string contains no whitespace characters.
// It returns an error if the string contains any whitespace, using the provided message and arguments.
func NoSpaces(s string, msg string, args ...any) error {
	if strings.IndexFunc(s, unicode.IsSpace) != -1 {
		return createError(msg, args...)
	}
	return nil
}

// NoNullChars checks if the provided string contains no null characters (ASCII 0).
// It returns an error if the string contains any null characters, using the provided message and arguments.
func NoNullChars(s string, msg string, args ...any) error {
	if strings.IndexByte(s, 0) != -1 {
		return createError(msg, args...)
	}
	return nil
}

// ContainsNoneOf checks if the provided string contains none of the characters in the given set.
// It returns an error if the string contains any of the specified characters, using the provided message and arguments.
func ContainsNoneOf(s string, chars string, msg string, args ...any) error {
	if strings.ContainsAny(s, chars) {
		return createError(msg, args...)
	}
	return nil
}

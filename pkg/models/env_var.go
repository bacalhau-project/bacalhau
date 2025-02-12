package models

import (
	"fmt"
	"strings"
	"unicode"
)

// EnvVarValue represents an environment variable value that can be
// either a literal value or a reference using prefix syntax (e.g., "env:VAR_NAME")
type EnvVarValue string

// Validate checks if the environment variable value has basic syntax
func (v EnvVarValue) Validate(name string) error {
	// Only validate that it's not empty
	if string(v) == "" {
		return fmt.Errorf("environment variable '%s' cannot have empty value", name)
	}
	return nil
}

// String implements fmt.Stringer interface
func (v EnvVarValue) String() string {
	return string(v)
}

// ValidateEnvVars validates a map of environment variables
func ValidateEnvVars(env map[string]EnvVarValue) error {
	for name, value := range env {
		if name == "" {
			return fmt.Errorf("environment variable name cannot be empty")
		}

		// Must be uppercase
		if name != strings.ToUpper(name) {
			return fmt.Errorf("environment variable '%s' must be uppercase", name)
		}

		// Must start with a letter
		if len(name) > 0 && !unicode.IsLetter(rune(name[0])) {
			return fmt.Errorf("environment variable '%s' must start with a letter", name)
		}

		// Only allow letters, numbers, and underscores
		for _, r := range name {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
				return fmt.Errorf("environment variable '%s' can only contain letters, numbers, and underscores", name)
			}
		}

		// Check reserved prefix
		if strings.HasPrefix(strings.ToUpper(name), EnvVarPrefix) {
			return fmt.Errorf("environment variable '%s' cannot start with %s", name, EnvVarPrefix)
		}

		if err := value.Validate(name); err != nil {
			return err
		}
	}
	return nil
}

// EnvVarsToStringMap converts a map of environment variables to a map of strings.
// This is useful when interfacing with APIs that expect traditional string-based environment variables.
func EnvVarsToStringMap(env map[string]EnvVarValue) map[string]string {
	if env == nil {
		return nil
	}
	result := make(map[string]string, len(env))
	for k, v := range env {
		result[k] = v.String()
	}
	return result
}

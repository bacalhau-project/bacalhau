package config

import (
	"fmt"
	"os"
	"strings"
)

// Getenv wraps os.Getenv and retrieves the value of the environment variable named by the config key.
// It returns the value, which will be empty if the variable is not present.
func Getenv(key string) string {
	return os.Getenv(KeyAsEnvVar(key))
}

// KeyAsEnvVar returns the environment variable corresponding to a config key
func KeyAsEnvVar(key string) string {
	return strings.ToUpper(
		fmt.Sprintf("%s_%s", environmentVariablePrefix, environmentVariableReplace.Replace(key)),
	)
}

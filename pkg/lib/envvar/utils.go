package envvar

import (
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/exp/maps"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func FromEnvVarValues(env map[string]models.EnvVarValue) map[string]string {
	envMap := make(map[string]string)
	for k, v := range env {
		envMap[k] = string(v)
	}
	return envMap
}

// ToSlice converts a map of environment variables to a slice of KEY=VALUE strings
func ToSlice(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}

	envSlice := make([]string, 0, len(env))
	for k, v := range env {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", k, v))
	}
	return envSlice
}

// FromSlice converts a slice of KEY=VALUE strings to a map of environment variables
func FromSlice(env []string) map[string]string {
	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	return envMap
}

// Sanitize converts a string to a valid environment variable value format
// by replacing invalid characters with underscores
func Sanitize(value string) string {
	return strings.Map(func(r rune) rune {
		// replace whitespace and '=' since they can cause problems
		if unicode.IsSpace(r) || r == '=' {
			return '_'
		}
		return r
	}, value)
}

// Merge combines environment variables from two sources.
// Values from priority map override values from base map.
func Merge(base, priority map[string]string) map[string]string {
	merged := maps.Clone(base)
	maps.Copy(merged, priority)
	return merged
}

// MergeSlices combines environment variables from two KEY=VALUE slice sources.
// Values from priority slice override values from base slice.
func MergeSlices(base, priority []string) []string {
	return ToSlice(Merge(FromSlice(base), FromSlice(priority)))
}

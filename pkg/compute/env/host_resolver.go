package env

import (
	"os"
	"regexp"
)

// HostResolver handles host environment variable references
type HostResolver struct {
	allowedPatterns []string
}

func NewHostResolver(allowList []string) *HostResolver {
	return &HostResolver{
		allowedPatterns: allowList,
	}
}

func (h *HostResolver) Prefix() string {
	return "env"
}

// Validate checks if the value is allowed
func (h *HostResolver) Validate(name string, value string) error {
	if !h.isAllowed(value) {
		return newErrNotAllowed(value)
	}
	return nil
}

// Value returns the value from host environment
func (h *HostResolver) Value(value string) (string, error) {
	if !h.isAllowed(value) {
		return "", newErrNotAllowed(value)
	}

	val, exists := os.LookupEnv(value)
	if !exists {
		return "", newErrNotFound(value)
	}

	return val, nil
}

func (h *HostResolver) isAllowed(varName string) bool {
	for _, pattern := range h.allowedPatterns {
		matched, err := regexp.MatchString(pattern, varName)
		if err == nil && matched {
			return true
		}
	}
	return false
}

package provider

import (
	"strings"
)

// sanitizeKey transforms the provider keys by:
// 1. Converting to lower case to make the matching case in-sensitive
// 2. Trim spaces
func sanitizeKey(key string) string {
	s := strings.TrimSpace(key)
	s = strings.ToLower(s)
	return s
}

package provider

import (
	"context"
	"strings"
)

// InstalledTypes returns all of the keys which the passed provider has
// installed.
func InstalledTypes[Value Providable](
	ctx context.Context,
	provider Provider[Value],
	allstrings []string,
) []string {
	var installedTypes []string
	for _, key := range allstrings {
		if provider.Has(ctx, key) {
			installedTypes = append(installedTypes, key)
		}
	}
	return installedTypes
}

// sanitizeKey transforms the provider keys by:
// 1. Converting to lower case to make the matching case in-sensitive
// 2. Trim spaces
func sanitizeKey(key string) string {
	s := strings.TrimSpace(key)
	s = strings.ToLower(s)
	return s
}

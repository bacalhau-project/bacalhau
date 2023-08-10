package provider

import (
	"context"
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

package env

import (
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
)

// ResolverMap handles delegation to specific environment variable resolvers
type ResolverMap struct {
	resolvers map[string]compute.EnvVarResolver
}

// ResolverParams contains configuration for environment variable resolvers
type ResolverParams struct {
	// AllowList specifies which host environment variables can be forwarded to jobs.
	// Supports glob patterns (e.g., "AWS_*", "API_*")
	AllowList []string
}

// NewResolver creates a new resolver map with configured resolvers
func NewResolver(params ResolverParams) *ResolverMap {
	hostResolver := NewHostResolver(params.AllowList)
	return &ResolverMap{
		resolvers: map[string]compute.EnvVarResolver{
			"env": hostResolver,
		},
	}
}

// Validate checks if any resolver can handle this value
func (m *ResolverMap) Validate(name string, value string) error {
	prefix, rest := parseValue(value)
	if prefix == "" {
		return nil // literal value
	}

	if resolver, exists := m.resolvers[prefix]; exists {
		return resolver.Validate(name, rest)
	}

	// Unknown prefix, treat as literal value
	return nil
}

// GetValue resolves a value using the appropriate resolver
func (m *ResolverMap) Value(value string) (string, error) {
	prefix, rest := parseValue(value)
	if prefix == "" {
		return value, nil // literal value
	}

	if resolver, exists := m.resolvers[prefix]; exists {
		return resolver.Value(rest)
	}

	// Unknown prefix, treat as literal value
	return value, nil
}

// parseValue extracts prefix and value parts
func parseValue(value string) (prefix string, rest string) {
	parts := strings.SplitN(value, PrefixDelimiter, 2)
	if len(parts) != 2 {
		return "", value
	}
	return parts[0], parts[1]
}

// compile-time check for interface implementation
var _ compute.EnvVarResolver = &ResolverMap{}

package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

// NewViperWithDefaultConfig create a viper instance to serve as a schema and load in the default configuration
// to our viper schema instance. This method is useful for inspecting default config values.
func NewViperWithDefaultConfig(cfg types.BacalhauConfig) *viper.Viper {
	viperSchema := viper.New()
	types.SetDefaults(cfg, types.WithViper(viperSchema))
	return viperSchema
}

// parseStringSliceToMap parses a slice of strings into a map.
// Each element in the slice should be a key-value pair in the form "key=value".
func parseStringSliceToMap(slice []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, item := range slice {
		key, value, err := flags.SeparatorParser("=")(item)
		if err != nil {
			return nil, fmt.Errorf("expected 'key=value', receieved invalid format for key-value pair: %s", item)
		}
		result[key] = value
	}
	return result, nil
}

func singleValueOrError(v ...string) (string, error) {
	if len(v) != 1 {
		return "", fmt.Errorf("expected single value got %d from %q", len(v), v)
	}
	return v[0], nil
}

// validNodeTypes returns true of the slice params contains strictly valid node types, false otherwise.
func validNodeTypes(slice []string) bool {
	// if nothing not valid
	if len(slice) == 0 {
		return false
	}
	// if there are more than 2 things its wrong
	if len(slice) > 2 {
		return false
	}
	// if there is only one thing it must be a compute or requester
	if len(slice) == 1 {
		if slice[0] != "compute" && slice[0] != "requester" {
			return false
		}
		return true
	}
	// there must be two things, one must be requester and the other must be compute
	return validNodeType(slice[0]) && validNodeType(slice[1])
}

// validNodeType returns true if t is requester or compute false otherwise.
func validNodeType(t string) bool {
	if t != "compute" && t != "requester" {
		return false
	}
	return true
}

// sanitizeKey transforms the provider keys by:
// 1. Converting to lower case to make the matching case in-sensitive
// 2. Trim spaces
func sanitizeKey(key string) string {
	s := strings.TrimSpace(key)
	s = strings.ToLower(s)
	return s
}

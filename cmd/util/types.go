package util

import (
	"fmt"
	"strings"
)

const null rune = 0

// ParseArrayAsMap processes a string array where each string looks like "k=v" and
// converts it into a map of string to string (in this case `m[k] = v`). In the
// process it makes sure that none of the values are ""
func ParseArrayAsMap(inputArray []string, fieldSep string) (map[string]string, error) {
	resultMap := make(map[string]string)

	for _, v := range inputArray {
		parts := strings.Split(v, fieldSep)
		if len(parts) != 2 {
			return nil, fmt.Errorf("expected field to be separate by an '=' character: %s", v)
		}

		key, value := parts[0], parts[1]

		// See wazero.ModuleConfig.WithEnv
		if value == "" || strings.ContainsRune(value, null) {
			return nil, fmt.Errorf("invalid environment variable %s=%s", key, value)
		}

		resultMap[key] = value
	}

	return resultMap, nil
}

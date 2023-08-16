package config

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// unmarshalCompositeKey takes a key and an output structure to unmarshal into. It gets the
// composite value associated with the given key and decodes it into the provided output structure.
// It's especially useful when the desired value is not directly associated with the key, but
// instead is spread across various nested sub-keys within the configuration.
func unmarshalCompositeKey(key string, output interface{}) error {
	compositeValue, err := getCompositeValue(key)
	if err != nil {
		return err
	}
	decoderConfig := &mapstructure.DecoderConfig{
		DecodeHook: mapstructure.TextUnmarshallerHookFunc(),
		Result:     output,
		TagName:    "mapstructure", // This is the default struct tag name used by Viper.
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return err
	}

	return decoder.Decode(compositeValue)
}

// getCompositeValue constructs a composite value for a given key. If the key directly corresponds
// to a set value in Viper, it returns that. Otherwise, it collects all nested values under that
// key and returns them as a nested map.
func getCompositeValue(key string) (map[string]interface{}, error) {
	var compositeValue map[string]interface{}

	// Fetch directly if the exact key exists
	if viper.IsSet(key) {
		rawValue := viper.Get(key)
		switch v := rawValue.(type) {
		case map[string]interface{}:
			compositeValue = v
		default:
			return map[string]interface{}{
				key: rawValue,
			}, nil
		}
	} else {
		compositeValue = make(map[string]interface{})
	}

	lowerKey := strings.ToLower(key)

	// Prepare a map for faster key lookup.
	viperKeys := viper.AllKeys()
	keyMap := make(map[string]string, len(viperKeys))
	for _, k := range viperKeys {
		keyMap[strings.ToLower(k)] = k
	}

	// Build a composite map of values for keys nested under the provided key.
	for lowerK, originalK := range keyMap {
		if strings.HasPrefix(lowerK, lowerKey+".") {
			parts := strings.Split(lowerK[len(lowerKey)+1:], ".")
			if err := setNested(compositeValue, parts, viper.Get(originalK)); err != nil {
				return nil, err
			}
		}
	}

	return compositeValue, nil
}

// setNested is a recursive helper function that sets a value in a nested map based on a slice of keys.
// It goes through each key, creating maps for each level as needed, and ultimately sets the value
// in the innermost map.
func setNested(m map[string]interface{}, keys []string, value interface{}) error {
	if len(keys) == 1 {
		m[keys[0]] = value
		return nil
	}

	// If the next map level doesn't exist, create it.
	if m[keys[0]] == nil {
		m[keys[0]] = make(map[string]interface{})
	}

	// Cast the nested level to a map and return an error if the type assertion fails.
	nestedMap, ok := m[keys[0]].(map[string]interface{})
	if !ok {
		return fmt.Errorf("key %s is not of type map[string]interface{}", keys[0])
	}

	return setNested(nestedMap, keys[1:], value)
}

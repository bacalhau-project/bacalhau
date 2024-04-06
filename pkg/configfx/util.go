package configfx

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

func getDefaultConfig(path string) types.BacalhauConfig {
	// derive the default config for the specified environment.
	defaultConfig := ForEnvironment()

	// set default values for path dependent config.
	defaultConfig.User.KeyPath = filepath.Join(path, UserPrivateKeyFileName)
	defaultConfig.User.Libp2pKeyPath = filepath.Join(path, Libp2pPrivateKeyFileName)
	defaultConfig.Node.ExecutorPluginPath = filepath.Join(path, PluginsPath)
	defaultConfig.Node.ComputeStoragePath = filepath.Join(path, ComputeStoragesPath)
	defaultConfig.Node.Compute.ExecutionStore.Path = filepath.Join(path, ComputeExecutionsStorePath)
	defaultConfig.Node.Requester.JobStore.Path = filepath.Join(path, OrchestratorJobStorePath)
	defaultConfig.Update.CheckStatePath = filepath.Join(path, UpdateCheckStatePath)
	defaultConfig.Auth.TokensPath = filepath.Join(path, TokensPath)

	// We default to the folder which contains the job store, and add
	// a subfolder for the network store.
	defaultConfig.Node.Network.StoreDir = filepath.Join(
		filepath.Dir(defaultConfig.Node.Requester.JobStore.Path),
		NetworkTransportStore,
	)

	return defaultConfig
}

// unmarshalCompositeKey takes a key and an output structure to unmarshal into. It gets the
// composite value associated with the given key and decodes it into the provided output structure.
// It's especially useful when the desired value is not directly associated with the key, but
// instead is spread across various nested sub-keys within the configuration.
func unmarshalCompositeKey(v *viper.Viper, key string, output interface{}) error {
	compositeValue, isNested, err := getCompositeValue(v, key)
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

	if isNested {
		val, ok := compositeValue[key]
		if !ok {
			// NB(forrest): this case should never happen as we ensure all configuration values
			// have a corresponding key via code gen. If this does occur it represents an error we need to debug.
			err := fmt.Errorf("CRITICAL ERROR: invalid configuration detected for key: %s. Config value not found", key)
			log.Err(err).Msg("invalid configuration detected")
			return err
		}
		return decoder.Decode(val)
	}

	return decoder.Decode(compositeValue)
}

// getCompositeValue constructs a composite value for a given key. If the key directly corresponds
// to a set value in Viper, it returns that, and false to indicate the value isn't nested under the key.
// Otherwise, it collects all nested values under that key and returns them as a nested map and true
// indicating the value is nested under the key.
func getCompositeValue(v *viper.Viper, key string) (map[string]interface{}, bool, error) {
	var compositeValue map[string]interface{}

	// Fetch directly if the exact key exists
	if v.IsSet(key) {
		rawValue := v.Get(key)
		switch v := rawValue.(type) {
		case map[string]interface{}:
			compositeValue = v
		default:
			return map[string]interface{}{
				key: rawValue,
			}, true, nil
		}
	} else {
		return nil, false, fmt.Errorf("configuration value not found for key: %s", key)
	}

	lowerKey := strings.ToLower(key)

	// Prepare a map for faster key lookup.
	viperKeys := v.AllKeys()
	keyMap := make(map[string]string, len(viperKeys))
	for _, k := range viperKeys {
		keyMap[strings.ToLower(k)] = k
	}

	// Build a composite map of values for keys nested under the provided key.
	for lowerK, originalK := range keyMap {
		if strings.HasPrefix(lowerK, lowerKey+".") {
			parts := strings.Split(lowerK[len(lowerKey)+1:], ".")
			if err := setNested(compositeValue, parts, v.Get(originalK)); err != nil {
				return nil, false, nil
			}
		}
	}

	return compositeValue, false, nil
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

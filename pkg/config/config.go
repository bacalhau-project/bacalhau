package config

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
)

const (
	environmentVariablePrefix = "BACALHAU"
	inferConfigTypes          = true
	automaticEnvVar           = true
)

var (
	environmentVariableReplace = strings.NewReplacer(".", "_")
	configDecoderHook          = viper.DecodeHook(mapstructure.TextUnmarshallerHookFunc())
)

func Init(defaultConfig *types.BacalhauConfig, path string, fileName, fileType string) (types.BacalhauConfig, error) {
	return initConfig(initParams{
		filePath:      path,
		fileName:      fileName,
		fileType:      fileType,
		fileHandler:   viper.SafeWriteConfig,
		defaultConfig: defaultConfig,
	})
}

func Load(path string, fileName, fileType string) (types.BacalhauConfig, error) {
	return initConfig(initParams{
		filePath:      path,
		fileName:      fileName,
		fileType:      fileType,
		fileHandler:   viper.ReadInConfig,
		defaultConfig: nil,
	})
}

func Get[T any](key string) (T, error) {
	raw := viper.Get(key)
	if raw == nil {
		return zeroValue[T](), fmt.Errorf("value not found for %s", key)
	}

	val, ok := raw.(T)
	if !ok {
		return zeroValue[T](), fmt.Errorf("value not of expected type, got: %T", raw)
	}

	return val, nil
}

func zeroValue[T any]() T {
	var zero T
	return zero
}

// Set sets the current configuration to `config`, useful for testing.
func Set(config types.BacalhauConfig) error {
	types.SetDefaults(config)
	return nil
}

// Reset clears all configuration, useful for testing.
func Reset() {
	viper.Reset()
}

// KeyAsEnvVar returns the environment variable corresponding to a config key
func KeyAsEnvVar(key string) string {
	return strings.ToUpper(
		fmt.Sprintf("%s_%s", environmentVariablePrefix, environmentVariableReplace.Replace(key)),
	)
}

type initParams struct {
	filePath      string
	fileName      string
	fileType      string
	fileHandler   func() error
	defaultConfig *types.BacalhauConfig
}

func initConfig(params initParams) (types.BacalhauConfig, error) {
	viper.AddConfigPath(params.filePath)
	viper.SetConfigName(params.fileName)
	viper.SetConfigType(params.fileType)
	viper.SetEnvPrefix(environmentVariablePrefix)
	viper.SetTypeByDefaultValue(inferConfigTypes)
	viper.SetEnvKeyReplacer(environmentVariableReplace)
	if params.defaultConfig != nil {
		if err := Set(*params.defaultConfig); err != nil {
			return types.BacalhauConfig{}, nil
		}
	}
	if err := params.fileHandler(); err != nil {
		return types.BacalhauConfig{}, err
	}
	if automaticEnvVar {
		viper.AutomaticEnv()
	}

	var out types.BacalhauConfig
	if err := viper.Unmarshal(&out, configDecoderHook); err != nil {
		return types.BacalhauConfig{}, err
	}

	// TODO this should be a part of the config.
	telemetry.SetupFromEnvs()

	return out, nil
}

// ForKey unmarshals configuration values associated with a given key into the provided cfg structure.
// It uses unmarshalCompositeKey internally to handle composite keys, ensuring values spread across
// nested sub-keys are correctly populated into the cfg structure.
//
// Parameters:
//   - key: The configuration key to retrieve values for.
//   - cfg: The structure into which the configuration values will be unmarshaled.
//
// Returns:
//   - An error if any occurred during unmarshaling; otherwise, nil.
func ForKey(key string, cfg interface{}) error {
	return unmarshalCompositeKey(key, cfg)
}

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

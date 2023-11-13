package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
)

func ClientAPIPort() uint16 {
	return uint16(viper.GetInt(types.NodeClientAPIPort))
}

func ClientAPIHost() string {
	return viper.GetString(types.NodeClientAPIHost)
}

func ServerAPIPort() uint16 {
	return uint16(viper.GetInt(types.NodeServerAPIPort))
}

func ServerAPIHost() string {
	return viper.GetString(types.NodeServerAPIHost)
}

func ServerAutoCertDomain() string {
	return viper.GetString(types.NodeServerAPITLSAutoCert)
}

func GetRequesterCertificateSettings() (string, string) {
	cert := viper.GetString(types.NodeServerAPITLSServerCertificate)
	key := viper.GetString(types.NodeServerAPITLSServerKey)
	return cert, key
}

func DevstackGetShouldPrintInfo() bool {
	return os.Getenv("DEVSTACK_PRINT_INFO") != ""
}

func DevstackSetShouldPrintInfo() {
	os.Setenv("DEVSTACK_PRINT_INFO", "1")
}

func DevstackEnvFile() string {
	return os.Getenv("DEVSTACK_ENV_FILE")
}

func ShouldKeepStack() bool {
	return os.Getenv("KEEP_STACK") != ""
}

type DockerCredentials struct {
	Username string
	Password string
}

func (d *DockerCredentials) IsValid() bool {
	return d.Username != "" && d.Password != ""
}

func GetDockerCredentials() DockerCredentials {
	return DockerCredentials{
		Username: os.Getenv("DOCKER_USERNAME"),
		Password: os.Getenv("DOCKER_PASSWORD"),
	}
}

func GetLibp2pConfig() (types.Libp2pConfig, error) {
	var libp2pCfg types.Libp2pConfig
	if err := ForKey(types.NodeLibp2p, &libp2pCfg); err != nil {
		return types.Libp2pConfig{}, err
	}
	return libp2pCfg, nil
}

func GetBootstrapPeers() ([]multiaddr.Multiaddr, error) {
	bootstrappers := viper.GetStringSlice(types.NodeBootstrapAddresses)
	peers := make([]multiaddr.Multiaddr, 0, len(bootstrappers))
	for _, peer := range bootstrappers {
		parsed, err := multiaddr.NewMultiaddr(peer)
		if err != nil {
			return nil, err
		}
		peers = append(peers, parsed)
	}
	return peers, nil
}

func GetLogMode() logger.LogMode {
	mode := viper.Get(types.NodeLoggingMode)
	switch v := mode.(type) {
	case logger.LogMode:
		return v
	case string:
		out, err := logger.ParseLogMode(v)
		if err != nil {
			log.Warn().Err(err).Msgf("invalid logging mode specified: %s", v)
		}
		return out
	default:
		log.Error().Msgf("unknown logging mode: %v", mode)
		return logger.LogModeDefault
	}
}

func GetAutoCertCachePath() string {
	return viper.GetString(types.NodeServerAPITLSAutoCertCachePath)
}

func GetLibp2pTracerPath() string {
	return viper.GetString(types.MetricsLibp2pTracerPath)
}

func GetEventTracerPath() string {
	return viper.GetString(types.MetricsEventTracerPath)
}

func GetExecutorPluginsPath() string {
	return viper.GetString(types.NodeExecutorPluginPath)
}

// TODO idk where this goes yet these are mostly random

func GetDownloadURLRequestRetries() int {
	return viper.GetInt(types.NodeDownloadURLRequestRetries)
}

func GetDownloadURLRequestTimeout() time.Duration {
	return viper.GetDuration(types.NodeDownloadURLRequestTimeout)
}

func SetVolumeSizeRequestTimeout(value time.Duration) {
	viper.Set(types.NodeVolumeSizeRequestTimeout, value)
}

func GetVolumeSizeRequestTimeout() time.Duration {
	return viper.GetDuration(types.NodeVolumeSizeRequestTimeout)
}

func GetUpdateCheckFrequency() time.Duration {
	return viper.GetDuration(types.UpdateCheckFrequency)
}

func GetStoragePath() string {
	path := viper.GetString(types.NodeComputeStoragePath)
	if path == "" {
		return os.TempDir()
	}
	return path
}

// PreferredAddress will allow for the specifying of
// the preferred address to listen on for cases where it
// is not clear, or where the address does not appear when
// using 0.0.0.0
func PreferredAddress() string {
	return os.Getenv("BACALHAU_PREFERRED_ADDRESS")
}

func GetStringMapString(key string) map[string]string {
	return viper.GetStringMapString(key)
}

func Get[T any](key string) (T, error) {
	raw := viper.Get(key)
	if raw == nil {
		return zeroValue[T](), fmt.Errorf("value not found for %s", key)
	}

	var val T
	val, ok := raw.(T)
	if !ok {
		err := ForKey(key, &val)
		if err != nil {
			return zeroValue[T](), fmt.Errorf("value not of expected type, got: %T", raw)
		}
	}

	return val, nil
}

func zeroValue[T any]() T {
	var zero T
	return zero
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
	compositeValue, isNested, err := getCompositeValue(key)
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
			return fmt.Errorf("CRITICAL ERROR: invalid configuration detected for key: %s. Config value not found", key)
		}
		return decoder.Decode(val)
	}

	return decoder.Decode(compositeValue)
}

// getCompositeValue constructs a composite value for a given key. If the key directly corresponds
// to a set value in Viper, it returns that, and false to indicate the value isn't nested under the key.
// Otherwise, it collects all nested values under that key and returns them as a nested map and true
// indicating the value is nested under the key.
func getCompositeValue(key string) (map[string]interface{}, bool, error) {
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
			}, true, nil
		}
	} else {
		return nil, false, fmt.Errorf("configuration value not found for key: %s", key)
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

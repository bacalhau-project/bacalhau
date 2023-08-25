package config

import (
	"os"
	"time"

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

func GetStoragePath() string {
	// TODO make this use the config when we figure out why our tests fail if this is anything but a tempDir
	// Currently there is a value in the config file for this but we are ignoring it.
	storagePath := os.Getenv("BACALHAU_STORAGE_PATH")
	if storagePath == "" {
		return os.TempDir()
	}
	return storagePath
}

// PreferredAddress will allow for the specificying of
// the preferred address to listen on for cases where it
// is not clear, or where the address does not appear when
// using 0.0.0.0
func PreferredAddress() string {
	return os.Getenv("BACALHAU_PREFERRED_ADDRESS")
}

func GetServeIPFSPath() string {
	return viper.GetString(types.NodeIPFSServePath)
}

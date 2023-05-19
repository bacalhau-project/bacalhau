package config

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog/log"
)

const (
	maxUInt16 uint16 = 0xFFFF
	minUInt16 uint16 = 0x0000
)

func DevstackShouldWriteEnvFile() bool {
	return DevstackEnvFile() != ""
}

func DevstackEnvFile() string {
	return os.Getenv("DEVSTACK_ENV_FILE")
}

func DevstackGetShouldPrintInfo() bool {
	return os.Getenv("DEVSTACK_PRINT_INFO") != ""
}

func DevstackSetShouldPrintInfo() {
	os.Setenv("DEVSTACK_PRINT_INFO", "1")
}

func ShouldKeepStack() bool {
	return os.Getenv("KEEP_STACK") != ""
}

func GetStoragePath() string {
	storagePath := os.Getenv("BACALHAU_STORAGE_PATH")
	if storagePath == "" {
		storagePath = os.TempDir()
	}
	return storagePath
}

func GetAPIHost() string {
	return os.Getenv("BACALHAU_HOST")
}

func GetAPIPort() *uint16 {
	portStr, found := os.LookupEnv("BACALHAU_PORT")
	if !found {
		return nil
	}

	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		panic(fmt.Sprintf("must be uint16 (%d-%d): %s", minUInt16, maxUInt16, portStr))
	}
	smallPort := uint16(port)
	return &smallPort
}

type contextKey int

const (
	getVolumeSizeRequestTimeoutKey contextKey = iota
)

const (
	// by default we wait 2 minutes for the IPFS network to resolve a CID
	// tests will override this using config.SetVolumeSizeRequestTimeout(2)
	getVolumeSizeRequestTimeout = 2 * time.Minute
)

// how long do we wait for a volume size request to timeout
// if a non-existing cid is asked for - the dockerIPFS.IPFSClient.GetCidSize(ctx, volume.Cid)
// function will hang for a long time - so we wrap that call in a timeout
// for tests - we only want to wait for 2 seconds because everything is on a local network
// in prod - we want to wait longer because we might be running a job that is
// using non-local CIDs
// the tests are expected to call SetVolumeSizeRequestTimeout to reduce this timeout
func GetVolumeSizeRequestTimeout(ctx context.Context) time.Duration {
	value := ctx.Value(getVolumeSizeRequestTimeoutKey)
	if value == nil {
		value = getVolumeSizeRequestTimeout
	}
	return value.(time.Duration)
}

func SetVolumeSizeRequestTimeout(ctx context.Context, value time.Duration) context.Context {
	return context.WithValue(ctx, getVolumeSizeRequestTimeoutKey, value)
}

// by default we wait 5 minutes for a URL to download
// tests will override this using config.SetDownloadURLRequestTimeoutSeconds(2)
var downloadURLRequestTimeoutSeconds int64 = 300

// how long do we wait for a URL to download
func GetDownloadURLRequestTimeout() time.Duration {
	return time.Duration(downloadURLRequestTimeoutSeconds) * time.Second
}

// how many times do we try to download a URL
var downloadURLRequestRetries = 3

// how long do we wait for a URL to download
func GetDownloadURLRequestRetries() int {
	return downloadURLRequestRetries
}

func GetLibp2pTracerPath() string {
	configPath := GetConfigPath()
	return filepath.Join(configPath, "bacalhau-libp2p-tracer.json")
}

func GetEventTracerPath() string {
	configPath := GetConfigPath()
	return filepath.Join(configPath, "bacalhau-event-tracer.json")
}

func GetConfigPath() string {
	suffix := ".bacalhau"
	env := os.Getenv("BACALHAU_PATH")
	var d string
	if env == "" {
		// e.g. /home/francesca/.bacalhau
		dirname, err := os.UserHomeDir()
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		d = filepath.Join(dirname, suffix)
	} else {
		// e.g. /data/.bacalhau
		d = filepath.Join(env, suffix)
	}
	// create dir if not exists
	if err := os.MkdirAll(d, util.OS_USER_RWX); err != nil {
		log.Fatal().Err(err).Send()
	}
	return d
}

const BitsForKeyPair = 2048

func GetPrivateKey(keyName string) (crypto.PrivKey, error) {
	configPath := GetConfigPath()

	// We include the port in the filename so that in devstack multiple nodes
	// running on the same host get different identities
	privKeyPath := filepath.Join(configPath, keyName)

	if _, err := os.Stat(privKeyPath); errors.Is(err, os.ErrNotExist) {
		// Private key does not exist - create and write it

		// Creates a new RSA key pair for this host.
		prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, BitsForKeyPair, rand.Reader)
		if err != nil {
			log.Error().Err(err)
			return nil, err
		}

		keyOut, err := os.OpenFile(privKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, util.OS_USER_RW)
		if err != nil {
			return nil, fmt.Errorf("failed to open key.pem for writing: %v", err)
		}
		privBytes, err := crypto.MarshalPrivateKey(prvKey)
		if err != nil {
			return nil, fmt.Errorf("unable to marshal private key: %v", err)
		}
		// base64 encode privBytes
		b64 := base64.StdEncoding.EncodeToString(privBytes)
		_, err = keyOut.WriteString(b64 + "\n")
		if err != nil {
			return nil, fmt.Errorf("failed to write to key file: %v", err)
		}
		if err := keyOut.Close(); err != nil {
			return nil, fmt.Errorf("error closing key file: %v", err)
		}
		log.Debug().Msgf("wrote %s", privKeyPath)
	}

	// Now that we've ensured the private key is written to disk, read it! This
	// ensures that loading it works even in the case where we've just created
	// it.

	// read the private key
	keyBytes, err := os.ReadFile(privKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %v", err)
	}
	// base64 decode keyBytes
	b64, err := base64.StdEncoding.DecodeString(string(keyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %v", err)
	}
	// parse the private key
	prvKey, err := crypto.UnmarshalPrivateKey(b64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	return prvKey, nil
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

// PreferredAddress will allow for the specificying of
// the preferred address to listen on for cases where it
// is not clear, or where the address does not appear when
// using 0.0.0.0
func PreferredAddress() string {
	return os.Getenv("BACALHAU_PREFERRED_ADDRESS")
}

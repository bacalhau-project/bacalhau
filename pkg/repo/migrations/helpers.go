package migrations

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"

	libp2p_crypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

const libp2pPrivateKey = "libp2p_private_key"

func getViper(r repo.FsRepo) (*viper.Viper, error) {
	repoPath, err := r.Path()
	if err != nil {
		return nil, err
	}

	configFile := filepath.Join(repoPath, config.FileName)
	v := viper.New()
	v.SetTypeByDefaultValue(true)
	v.SetConfigFile(configFile)

	// read existing config file if it exists
	if err := v.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	return v, nil
}

func configExists(r repo.FsRepo) (bool, error) {
	repoPath, err := r.Path()
	if err != nil {
		return false, err
	}

	configFile := filepath.Join(repoPath, config.FileName)
	_, err = os.Stat(configFile)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func readConfig(r repo.FsRepo) (*viper.Viper, types.BacalhauConfig, error) {
	v, err := getViper(r)
	if err != nil {
		return nil, types.BacalhauConfig{}, err
	}
	var fileCfg types.BacalhauConfig
	if err := v.Unmarshal(&fileCfg, config.DecoderHook); err != nil {
		return v, types.BacalhauConfig{}, fmt.Errorf("failed to unmarshal config file: %w", err)
	}
	return v, fileCfg, nil
}

// TODO: remove this with 1.5 release as we don't expect users to migrate from 1.2 to 1.5
func getLibp2pNodeID(repoPath string) (string, error) {
	path := filepath.Join(repoPath, libp2pPrivateKey)
	privKey, err := loadLibp2pPrivKey(path)
	if err != nil {
		return "", err
	}
	peerID, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		return "", err
	}
	return peerID.String(), nil
}

func loadLibp2pPrivKey(path string) (libp2p_crypto.PrivKey, error) {
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}
	// base64 decode keyBytes
	b64, err := base64.StdEncoding.DecodeString(string(keyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}
	// parse the private key
	key, err := libp2p_crypto.UnmarshalPrivateKey(b64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	return key, nil
}

// copyFile copies a file from srcPath to dstPath, preserving the file permissions.
// It opens the source file, creates the destination file, copies the content,
// and sets the destination file's permissions to match the source file. It also
// ensures that the destination file's contents are flushed to disk before returning.
func copyFile(srcPath, dstPath string) error {
	// Open the source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Get the file info of the source file to retrieve its permissions
	srcFileInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	// Create the destination file
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy the contents from source to destination
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	// Set the destination file's permissions to match the source file's permissions
	err = os.Chmod(dstPath, srcFileInfo.Mode())
	if err != nil {
		return err
	}

	// Flush the contents to disk
	err = dstFile.Sync()
	if err != nil {
		return err
	}

	return nil
}

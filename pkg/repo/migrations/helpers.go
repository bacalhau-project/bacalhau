package migrations

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

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

// haveSameElements returns true if arr1 and arr2 have the same elements, false otherwise.
func haveSameElements(arr1, arr2 []string) bool {
	if len(arr1) != len(arr2) {
		return false
	}

	elementCount := make(map[string]int)

	for _, item := range arr1 {
		elementCount[item]++
	}

	for _, item := range arr2 {
		if count, exists := elementCount[item]; !exists || count == 0 {
			return false
		}
		elementCount[item]--
	}

	return true
}

func getLibp2pNodeID(path string) (string, error) {
	privKey, err := config.GetLibp2pPrivKey(path)
	if err != nil {
		return "", err
	}
	peerID, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		return "", err
	}
	return peerID.String(), nil
}

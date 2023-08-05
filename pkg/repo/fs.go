package repo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/config_v2"
)

// cribbed from lotus

type FsRepo struct {
	path       string
	configPath string
}

const fsConfigType = "toml"
const fsConfigName = "config"

func NewFS(path string) (*FsRepo, error) {
	path, err := homedir.Expand(path)
	if err != nil {
		return nil, err
	}

	return &FsRepo{
		path:       path,
		configPath: filepath.Join(path, fmt.Sprintf("%s.%s", fsConfigName, fsConfigType)),
	}, nil

}

func (fsr *FsRepo) Exists() (bool, error) {
	_, err := os.Stat(fsr.path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (fsr *FsRepo) Init(cfg config_v2.BacalhauConfig) error {
	exist, err := fsr.Exists()
	if err != nil {
		return err
	}
	if exist {
		log.Info().Msgf("Repo found at '%s", fsr.path)
		return nil
	}

	log.Info().Msgf("Initializing repo at '%s'", fsr.path)
	// 0755 The owner can read, write, and execute, while others can read and execute.
	err = os.MkdirAll(fsr.path, 0755) //nolint: gosec
	if err != nil && !os.IsExist(err) {
		return err
	}

	if err := config_v2.InitConfig(fsr.path, fsConfigName, fsConfigType); err != nil {
		return err
	}

	return nil
}

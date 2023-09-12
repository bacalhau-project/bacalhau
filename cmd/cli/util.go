package cli

import (
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

const defaultBacalhauDir = ".bacalhau"

func defaultRepo() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Debug().Err(err).Msg("failed to get user home dir as default repo. Must set --repo flag or BACALHAU_DIR to specify a repo")
		return ""
	}
	return filepath.Join(home, defaultBacalhauDir)
}

package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

const defaultBacalhauDir = ".bacalhau"

func defaultRepo() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home dir: %w", err)
	}
	return filepath.Join(home, defaultBacalhauDir), nil
}

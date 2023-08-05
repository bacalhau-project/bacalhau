package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

const DefaultBacalhauDir = ".bacalhau"

func DefaultRepo() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home dir: %w", err)
	}
	return filepath.Join(home, DefaultBacalhauDir), nil
}

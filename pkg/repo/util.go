package repo

import (
	"fmt"
	"os"
)

func dirExists(path string) (bool, error) {
	// stat path to check if exists
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		// it exists, but we failed to stat it, return err
		return false, fmt.Errorf("failed to check if directory exists at path %q: %w", path, err)
	}
	// if the path exists, ensure it's a directory
	if !stat.IsDir() {
		return false, fmt.Errorf("path %q is a file, expected directory", path)
	}
	return true, nil
}

func fileExists(path string) (bool, error) {
	// Check if the file exists
	_, err := os.Stat(path)
	if err == nil {
		// File exists
		return true, nil
	} else if !os.IsNotExist(err) {
		// os.Stat returned an error other than "file does not exist"
		return false, fmt.Errorf("failed to check if file exists at path: %w", err)
	}
	// file does not exist
	return false, nil
}

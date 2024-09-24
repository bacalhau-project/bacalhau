package validate

import (
	"errors"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"unicode/utf8"
)

// IsValidPath checks if the given path is valid according to the current operating system's rules.
// It performs the following checks:
// 1. Ensures the path contains valid UTF-8 characters.
// 2. Verifies that the path is absolute (starts with '/').
// 3. Applies OS-specific validation:
//   - For Windows: Checks drive letter or UNC format, component length, reserved names,
//     invalid characters, and total path length.
//   - For Unix-like systems: Checks for null bytes and forward slashes within components.
//
// Returns nil if the path is valid, or an error describing the first validation failure encountered.
func IsValidPath(path string) error {
	// Check for valid UTF-8 encoding
	if !utf8.ValidString(path) {
		return errors.New("path contains invalid UTF-8 characters")
	}

	// Check if the path is absolute (starts with '/')
	if !filepath.IsAbs(path) {
		return errors.New("path must be absolute (start with '/')")
	}

	// Handle OS-specific forbidden characters
	if runtime.GOOS == "windows" {
		if err := validateWindowsPath(path); err != nil {
			return err
		}
	} else {
		if err := validateUnixPath(path); err != nil {
			return err
		}
	}

	return nil
}

func validateUnixPath(path string) error {
	// Check if the path is empty
	if path == "" {
		return errors.New("path is empty")
	}

	// Split the path into components
	components := strings.Split(path, "/")

	for _, component := range components {
		// Skip empty components (occurs for root '/' and double slashes '//')
		if component == "" {
			continue
		}

		// Check if the component is "." or ".."
		if component == "." || component == ".." {
			continue
		}

		// Check if the component contains null byte
		if strings.Contains(component, "\x00") {
			return errors.New("path component cannot contain null byte: " + component)
		}

		// Check if the component contains forward slash
		if strings.Contains(component, "/") {
			return errors.New("path component cannot contain forward slash: " + component)
		}
	}

	return nil
}

func validateWindowsPath(path string) error {
	// Check if the path is empty
	if path == "" {
		return errors.New("path is empty")
	}

	// Split the path into components
	components := strings.Split(path, "\\")

	// Regular expression to match valid Windows filename characters
	validName := regexp.MustCompile(`^[^<>:"/\\|?*\x00-\x1F]*[^<>:"/\\|?*\x00-\x1F\. ]$`)

	// Check drive letter or UNC path
	if len(components) > 0 {
		if len(components[0]) == 2 && components[0][1] == ':' {
			// Drive letter path
			if !regexp.MustCompile(`^[A-Za-z]:$`).MatchString(components[0]) {
				return errors.New("invalid drive letter")
			}
		} else if components[0] == "" && len(components) > 1 && components[1] == "" {
			// UNC path
			if len(components) < 4 {
				return errors.New("invalid UNC path")
			}
		} else {
			return errors.New("path must start with a drive letter or be a UNC path")
		}
	}

	for i, component := range components {
		// Skip empty components and the first two components of a UNC path
		if component == "" || (i < 2 && strings.HasPrefix(path, "\\\\")) {
			continue
		}

		// Check if the component is "." or ".."
		if component == "." || component == ".." {
			continue
		}

		// Check if the component contains only valid characters
		if !validName.MatchString(component) {
			return errors.New("invalid characters in path component: " + component)
		}

		// Check reserved names
		reservedNames := []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}
		upperComponent := strings.ToUpper(component)
		for _, name := range reservedNames {
			if upperComponent == name || strings.HasPrefix(upperComponent, name+".") {
				return errors.New("path component cannot be a reserved name: " + component)
			}
		}

		// Check length
		if len(component) > 255 {
			return errors.New("path component is too long: " + component)
		}
	}

	// Check total path length
	if len(path) > 32767 {
		return errors.New("path is too long")
	}

	return nil
}

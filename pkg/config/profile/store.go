package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
)

const (
	profileExtension = ".yaml"
	currentSymlink   = "current"
)

// Store handles profile persistence operations.
type Store struct {
	dir string
}

// NewStore creates a new profile store at the given directory.
func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

// ensureDir creates the profiles directory if it doesn't exist.
func (s *Store) ensureDir() error {
	return os.MkdirAll(s.dir, 0755)
}

// validateName checks that a profile name is valid.
func validateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("profile name cannot be empty")
	}
	return nil
}

// sanitizeName sanitizes profile names to prevent path traversal attacks.
func sanitizeName(name string) string {
	// Use filepath.Base to extract just the filename portion,
	// which handles path separators for the current OS
	base := filepath.Base(name)
	// Replace any remaining ".." sequences
	base = strings.ReplaceAll(base, "..", "_")
	return base
}

// profilePath returns the file path for a profile.
func (s *Store) profilePath(name string) string {
	return filepath.Join(s.dir, sanitizeName(name)+profileExtension)
}

// currentPath returns the path to the current symlink.
func (s *Store) currentPath() string {
	return filepath.Join(s.dir, currentSymlink)
}

// Save saves a profile to disk.
func (s *Store) Save(name string, p *Profile) error {
	if err := validateName(name); err != nil {
		return err
	}
	if err := p.Validate(); err != nil {
		return fmt.Errorf("invalid profile: %w", err)
	}
	if err := s.ensureDir(); err != nil {
		return fmt.Errorf("creating profiles directory: %w", err)
	}

	data, err := yaml.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshaling profile: %w", err)
	}

	path := s.profilePath(name)
	if err := os.WriteFile(path, data, util.OS_USER_RW); err != nil {
		return fmt.Errorf("writing profile file: %w", err)
	}

	return nil
}

// Load loads a profile from disk.
func (s *Store) Load(name string) (*Profile, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}
	path := s.profilePath(name)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("profile %q not found", name)
		}
		return nil, fmt.Errorf("reading profile file: %w", err)
	}

	var p Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing profile: %w", err)
	}

	return &p, nil
}

// Delete removes a profile from disk.
func (s *Store) Delete(name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	// Check if this is the current profile
	current, _ := s.GetCurrent()
	if current == name {
		// Remove the current symlink
		os.Remove(s.currentPath())
	}

	path := s.profilePath(name)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("profile %q not found", name)
		}
		return fmt.Errorf("deleting profile: %w", err)
	}

	return nil
}

// List returns all profile names.
func (s *Store) List() ([]string, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("reading profiles directory: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == currentSymlink {
			continue
		}
		if profileName, found := strings.CutSuffix(name, profileExtension); found {
			names = append(names, profileName)
		}
	}

	return names, nil
}

// SetCurrent sets the current profile via symlink.
func (s *Store) SetCurrent(name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	// Verify profile exists
	if _, err := s.Load(name); err != nil {
		return err
	}

	currentPath := s.currentPath()
	// Remove existing symlink if present
	os.Remove(currentPath)

	// Create relative symlink
	target := sanitizeName(name) + profileExtension
	if err := os.Symlink(target, currentPath); err != nil {
		return fmt.Errorf("creating current symlink: %w", err)
	}

	return nil
}

// GetCurrent returns the name of the current profile, or empty string if none.
func (s *Store) GetCurrent() (string, error) {
	currentPath := s.currentPath()
	target, err := os.Readlink(currentPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("reading current symlink: %w", err)
	}

	// Extract name from target (e.g., "prod.yaml" -> "prod")
	name := strings.TrimSuffix(target, profileExtension)
	return name, nil
}

// Exists checks if a profile exists.
func (s *Store) Exists(name string) bool {
	if validateName(name) != nil {
		return false
	}
	_, err := os.Stat(s.profilePath(name))
	return err == nil
}

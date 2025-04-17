package licensing

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/bacalhau-project/bacalhau/pkg/lib/license"
)

// reader handles loading and reading license data
type reader struct {
	path            string
	rawLicenseToken string
	licenseClaims   *license.LicenseClaims
}

// NewReader creates and initializes a new reader
func NewReader(path string) (Reader, error) {
	loader := &reader{
		path: path,
	}

	// If no path provided, return empty loader
	if path == "" {
		return loader, nil
	}

	// Try to read and validate the license
	if err := loader.load(); err != nil {
		return nil, fmt.Errorf("failed to load license: %w", err)
	}

	return loader, nil
}

// load reads and validates the license file
func (l *reader) load() error {
	// Read the license file
	licenseData, err := os.ReadFile(l.path)
	if err != nil {
		return fmt.Errorf("failed to read license file: %w", err)
	}

	var licenseFile LicenseFile
	if err = json.Unmarshal(licenseData, &licenseFile); err != nil {
		return fmt.Errorf("failed to parse license file: %w", err)
	}

	// Validate the license token
	validator, err := license.NewOfflineLicenseValidator()
	if err != nil {
		return fmt.Errorf("failed to create license validator: %w", err)
	}

	claims, err := validator.Validate(licenseFile.License)
	if err != nil {
		return err
	}

	l.rawLicenseToken = licenseFile.License
	l.licenseClaims = claims
	return nil
}

// License returns the current license claims
func (l *reader) License() *license.LicenseClaims {
	return l.licenseClaims
}

// RawToken returns the raw license token
func (l *reader) RawToken() string {
	return l.rawLicenseToken
}

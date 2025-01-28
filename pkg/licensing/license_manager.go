package licensing

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/lib/license"
)

// LicenseManager handles license management and validation for the node
type LicenseManager struct {
	licenseConfig   *types.License
	rawLicenseToken string
	licenseClaims   *license.LicenseClaims
}

// NewLicenseManager creates and initializes a new LicenseManager
func NewLicenseManager(config *types.License) (*LicenseManager, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	licenseValidator, err := license.NewOfflineLicenseValidator()
	if err != nil {
		return nil, fmt.Errorf("failed to create license validator: %w", err)
	}

	licenseFilePath := config.LocalPath

	// Return a LicenseManager with no claims if no file path was defined
	if licenseFilePath == "" {
		return &LicenseManager{
			licenseConfig:   config,
			licenseClaims:   nil,
			rawLicenseToken: "",
		}, nil
	}

	// Try to read the license file, fail if the file is not found or malformed
	licenseData, err := os.ReadFile(licenseFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read license file: %w", err)
	}

	// Verify the JSON structure of the license file
	var licenseFile struct {
		License string `json:"license"`
	}
	if err = json.Unmarshal(licenseData, &licenseFile); err != nil {
		return nil, fmt.Errorf("failed to parse license file: %w", err)
	}

	rawLicenseToken := licenseFile.License
	licenseClaims, err := licenseValidator.Validate(rawLicenseToken)

	if err != nil {
		return nil, err
	}

	return &LicenseManager{
		licenseConfig:   config,
		licenseClaims:   licenseClaims,
		rawLicenseToken: rawLicenseToken,
	}, nil
}

// License validates the current license token and returns the license claims and if it is expired
func (l *LicenseManager) License() *license.LicenseClaims {
	return l.licenseClaims
}

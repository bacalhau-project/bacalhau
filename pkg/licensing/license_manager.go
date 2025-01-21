package licensing

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/lib/license"
)

// LicenseManager handles license management and validation for the node
type LicenseManager struct {
	bacalhauConfig    *types.Bacalhau
	licenseConfigured bool
	licenseToken      string
	licenseValidator  *license.LicenseValidator
}

// NewLicenseManager creates and initializes a new LicenseManager
func NewLicenseManager(config *types.Bacalhau) (*LicenseManager, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	validator, err := license.NewOfflineLicenseValidator()
	if err != nil {
		return nil, fmt.Errorf("failed to create license validator: %w", err)
	}

	licensePath := config.Orchestrator.License.LocalPath
	licenseConfigured := licensePath != ""

	var licenseToken string
	if licenseConfigured {
		// Try to read the license file, fail if the file is not found or malformed
		// Will not inspect the license itself.
		licenseData, err := os.ReadFile(licensePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read license file: %w", err)
		}

		// Verify the JSON structure
		var licenseFile struct {
			License string `json:"license"`
		}
		if err := json.Unmarshal(licenseData, &licenseFile); err != nil {
			return nil, fmt.Errorf("failed to parse license file: %w", err)
		}
		licenseToken = licenseFile.License
	}

	return &LicenseManager{
		bacalhauConfig:    config,
		licenseConfigured: licenseConfigured,
		licenseToken:      licenseToken,
		licenseValidator:  validator,
	}, nil
}

// ValidateLicense validates the current license token and returns the license claims
func (l *LicenseManager) ValidateLicense() (*license.LicenseClaims, error) {
	if !l.licenseConfigured {
		return nil, fmt.Errorf("no license configured for orchestrator")
	}

	claims, err := l.licenseValidator.ValidateToken(l.licenseToken)
	if err != nil {
		return nil, fmt.Errorf("invalid license: %w", err)
	}

	return claims, nil
}

// ValidateLicenseWithNodeCount validates the license and checks if the number of nodes is within the licensed limit
func (l *LicenseManager) ValidateLicenseWithNodeCount(nodeCount int) (*license.LicenseClaims, error) {
	claims, err := l.ValidateLicense()
	if err != nil {
		return nil, err
	}

	// Get max_nodes from capabilities
	maxNodesStr, exists := claims.Capabilities["max_nodes"]
	if !exists {
		return nil, fmt.Errorf("license does not specify max_nodes capability")
	}

	maxNodes, err := strconv.Atoi(maxNodesStr)
	if err != nil {
		return nil, fmt.Errorf("invalid max_nodes value in license: %w", err)
	}

	if nodeCount > maxNodes {
		return nil, fmt.Errorf("node count %d exceeds licensed limit of %d nodes", nodeCount, maxNodes)
	}

	return claims, nil
}

package licensing

import (
	"fmt"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/lib/license"
)

type Manager interface {
	// License returns the current license claims
	License() *license.LicenseClaims

	// Validate returns the current license validation state
	Validate() LicenseValidationState

	// Start starts the validation loop
	Start()

	// Stop stops the validation loop
	Stop()
}

type Reader interface {
	// License returns the current license claims
	License() *license.LicenseClaims

	// RawToken returns the raw license token
	RawToken() string
}

// LicenseValidationState represents the current state of license validation
type LicenseValidationState struct {
	// Type indicates the type of validation state
	Type LicenseValidationType
	// Message provides a human-readable message about the state
	Message string
}

// LicenseValidationType represents the type of license validation state
type LicenseValidationType int

const (
	// LicenseValidationTypeValid indicates the license is valid and within limits
	LicenseValidationTypeValid LicenseValidationType = iota
	// LicenseValidationTypeNoLicense indicates no license is configured
	LicenseValidationTypeNoLicense
	// LicenseValidationTypeExpired indicates the license has expired
	LicenseValidationTypeExpired
	// LicenseValidationTypeExceededNodes indicates the number of connected nodes exceeds the license limit
	LicenseValidationTypeExceededNodes
	// LicenseValidationTypeFreeTierExceeded indicates the number of connected nodes exceeds the free tier limit
	LicenseValidationTypeFreeTierExceeded
	// LicenseValidationTypeFreeTierValid indicates the system is within free tier limits
	LicenseValidationTypeFreeTierValid
	// LicenseValidationTypeSkipped indicates that license validation is skipped
	LicenseValidationTypeSkipped
	// licenseValidationTypeUnknown indicates an unknown validation state
	licenseValidationTypeUnknown
)

// String returns the string representation of the validation type
func (t LicenseValidationType) String() string {
	switch t {
	case LicenseValidationTypeValid:
		return "Valid"
	case LicenseValidationTypeNoLicense:
		return "NoLicense"
	case LicenseValidationTypeExpired:
		return "Expired"
	case LicenseValidationTypeExceededNodes:
		return "ExceededNodes"
	case LicenseValidationTypeFreeTierExceeded:
		return "FreeTierExceeded"
	case LicenseValidationTypeFreeTierValid:
		return "FreeTierValid"
	case LicenseValidationTypeSkipped:
		return "Skipped"
	default:
		return "Unknown"
	}
}

// ParseLicenseValidationType parses a string into a LicenseValidationType
func ParseLicenseValidationType(s string) (LicenseValidationType, error) {
	for typ := LicenseValidationTypeValid; typ <= licenseValidationTypeUnknown; typ++ {
		if strings.EqualFold(typ.String(), strings.TrimSpace(s)) {
			return typ, nil
		}
	}
	return licenseValidationTypeUnknown, fmt.Errorf("%T: unknown type '%s'", licenseValidationTypeUnknown, s)
}

// MarshalText implements the encoding.TextMarshaler interface
func (t LicenseValidationType) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface
func (t *LicenseValidationType) UnmarshalText(text []byte) error {
	typ, err := ParseLicenseValidationType(string(text))
	if err != nil {
		return err
	}
	*t = typ
	return nil
}

func (t LicenseValidationType) IsValid() bool {
	return t == LicenseValidationTypeValid || t == LicenseValidationTypeFreeTierValid || t == LicenseValidationTypeSkipped
}

// LicenseState represents the current state of the license
type LicenseState struct {
	Type    string `json:"Type"`
	Message string `json:"Message"`
}

// LicenseFile is the structure of the license file
type LicenseFile struct {
	License string `json:"license"`
}

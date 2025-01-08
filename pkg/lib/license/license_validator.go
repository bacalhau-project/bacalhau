package license

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// LicenseValidator handles JWT validation with JWKS support
type LicenseValidator struct {
	keyFunc jwt.Keyfunc
}

// LicenseClaims represents the expected claims in your license token
type LicenseClaims struct {
	jwt.RegisteredClaims
	// Add your custom license claims here
	Product        string            `json:"product,omitempty"`
	LicenseID      string            `json:"license_id,omitempty"`
	LicenseType    string            `json:"license_type,omitempty"`
	LicenseVersion string            `json:"license_version,omitempty"`
	CustomerID     string            `json:"customer_id,omitempty"`
	Capabilities   map[string]string `json:"capabilities,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// NewLicenseValidatorFromFile creates a new validator from a JWKS file
func NewLicenseValidatorFromFile(jwksPath string) (*LicenseValidator, error) {
	jwksData, err := os.ReadFile(jwksPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read JWKS file: %w", err)
	}

	return NewLicenseValidatorFromJSON(jwksData)
}

// NewLicenseValidatorFromJSON creates a new validator from JWKS JSON
func NewLicenseValidatorFromJSON(jwksJSON json.RawMessage) (*LicenseValidator, error) {
	if len(jwksJSON) == 0 {
		return nil, fmt.Errorf("empty JWKS JSON")
	}

	// Parse the JSON first to validate structure
	var jwks struct {
		Keys []interface{} `json:"keys"`
	}
	if err := json.Unmarshal(jwksJSON, &jwks); err != nil {
		return nil, fmt.Errorf("invalid JWKS JSON: %w", err)
	}

	// Check if keys array exists and is not empty
	if jwks.Keys == nil {
		return nil, fmt.Errorf("missing 'keys' array in JWKS")
	}
	if len(jwks.Keys) == 0 {
		return nil, fmt.Errorf("empty 'keys' array in JWKS")
	}

	// Create the JWKS key function
	keyFunc, err := keyfunc.NewJWKSetJSON(jwksJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWKS key function: %w", err)
	}

	return &LicenseValidator{
		keyFunc: keyFunc.Keyfunc,
	}, nil
}

// ValidateToken validates a license token and returns the claims
func (v *LicenseValidator) ValidateToken(tokenString string) (*LicenseClaims, error) {
	var claims LicenseClaims

	// Parse and validate the token
	token, err := jwt.ParseWithClaims(tokenString, &claims, v.keyFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Additional validation can be added here
	if err := v.validateAdditionalConstraints(&claims); err != nil {
		return nil, err
	}

	return &claims, nil
}

// validateAdditionalConstraints performs additional business logic validation
func (v *LicenseValidator) validateAdditionalConstraints(claims *LicenseClaims) error {
	now := time.Now()

	// Check if token is expired
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(now) {
		return fmt.Errorf("license has expired")
	}

	// Check if token is not yet valid
	if claims.NotBefore != nil && claims.NotBefore.After(now) {
		return fmt.Errorf("license is not yet valid")
	}

	// Only perform additional validations for v1 licenses
	if claims.LicenseVersion == "v1" {
		// Verify product name
		if claims.Product != "Bacalhau" {
			return fmt.Errorf("invalid product: expected 'Bacalhau', got '%s'", claims.Product)
		}

		// Verify required fields are not empty
		if claims.LicenseID == "" {
			return fmt.Errorf("license_id is required")
		}
		if claims.LicenseType == "" {
			return fmt.Errorf("license_type is required")
		}
		if claims.CustomerID == "" {
			return fmt.Errorf("customer_id is required")
		}
		if claims.Subject == "" {
			return fmt.Errorf("subject is required")
		}
		if claims.ID == "" {
			return fmt.Errorf("jti is required")
		}

		// Verify issuer
		if claims.Issuer != "https://expanso.io/" {
			return fmt.Errorf("invalid issuer: expected 'https://expanso.io/', got '%s'", claims.Issuer)
		}
	} else {
		return fmt.Errorf("unsupported license version: %s", claims.LicenseVersion)
	}

	return nil
}

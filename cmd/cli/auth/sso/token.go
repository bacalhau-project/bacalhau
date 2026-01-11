package sso

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"

	"github.com/bacalhau-project/bacalhau/cmd/util"
)

// SSOTokenOptions is a struct to support SSO Token functionality
type SSOTokenOptions struct {
	Decode bool // Decode JWT Token
}

// NewSSOTokenOptions returns initialized SSOTokenOptions
func NewSSOTokenOptions() *SSOTokenOptions {
	return &SSOTokenOptions{
		Decode: false,
	}
}

func NewSSOTokenCmd() *cobra.Command {
	o := NewSSOTokenOptions()
	tokenCmd := &cobra.Command{
		Use:   "token",
		Short: "Show current environment SSO JWT token",
		Long:  `Show current environment SSO JWT token if present.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := util.SetupRepoConfig(cmd)
			if err != nil {
				return bacerrors.New("failed to setup bacalhau repository config")
			}
			return o.runSSOToken(cmd, cfg)
		},
	}

	// Add force flag to skip confirmation
	tokenCmd.Flags().BoolVarP(&o.Decode, "decode", "d", false, "Decode JWT Token")

	return tokenCmd
}

// runSSOToken handles the SSO token command
func (o *SSOTokenOptions) runSSOToken(cmd *cobra.Command, cfg types.Bacalhau) error {
	apiURL, _ := util.ConstructAPIEndpoint(cfg.API)

	authTokenPath, err := cfg.JWTTokensPath()
	if err != nil {
		return bacerrors.New("unable to find local SSO session file")
	}

	existingCred, readErr := util.ReadToken(authTokenPath, apiURL)
	if readErr != nil {
		return bacerrors.New("unable to retrieve saved SSO credentials")
	}

	if existingCred == nil {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No authentication token found")
		return nil
	}

	if o.Decode {
		// Parse the token without verification
		token, err := jwt.Parse(existingCred.Value, func(token *jwt.Token) (interface{}, error) {
			return nil, nil // We're not verifying, just decoding
		})
		if err != nil && !strings.Contains(err.Error(), "key is of invalid type") {
			return bacerrors.Newf("failed to parse token: %s", err)
		}

		// Pretty print the header
		headerJSON, err := json.MarshalIndent(token.Header, "", "  ")
		if err != nil {
			return bacerrors.Newf("failed to marshal header: %s", err)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Header:")
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(headerJSON))

		// Pretty print the claims
		claimsJSON, err := json.MarshalIndent(token.Claims, "", "  ")
		if err != nil {
			return bacerrors.Newf("failed to marshal claims: %s", err)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nClaims:")
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(claimsJSON))
	} else {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), existingCred.Value)
	}

	return nil
}

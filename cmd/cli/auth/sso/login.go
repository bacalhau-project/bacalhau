package sso

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/sso"
)

const errorHint = "Set LOG_LEVEL=DEBUG and re-run for detailed logs"

// SSOOptions is a struct to support node command
type SSOLoginOptions struct{}

// NewSSOLoginOptions returns initialized Options
func NewSSOLoginOptions() *SSOLoginOptions {
	return &SSOLoginOptions{}
}

func NewSSOLoginCmd() *cobra.Command {
	o := NewSSOLoginOptions()
	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Login using SSO",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// initialize a new or open an existing repo merging any config file(s) it contains into cfg.
			cfg, err := util.SetupRepoConfig(cmd)
			if err != nil {
				log.Debug().Err(err).Msg("failed to setup bacalhau repository config")
				return bacerrors.New("failed to setup bacalhau repository config").
					WithHint(errorHint)
			}

			api, err := util.NewAPIClientManager(cmd, cfg).GetUnauthenticatedAPIClient()
			if err != nil {
				log.Debug().Err(err).Msg("failed to initialize API client")
				return bacerrors.New("failed to fetch supported authentication details from server").
					WithHint(errorHint)
			}
			return o.runSSOLogin(cmd, api, cfg)
		},
	}

	return loginCmd
}

// Run executes node command
func (o *SSOLoginOptions) runSSOLogin(cmd *cobra.Command, api client.API, cfg types.Bacalhau) error {
	ctx := cmd.Context()

	apiURL, urlScheme := util.ConstructAPIEndpoint(cfg.API)
	authTokenPath, err := cfg.JWTTokensPath()
	if err != nil {
		log.Debug().Err(err).Msg("failed to get JWTTokensPath path")
		return bacerrors.New("unable to save temporary SSO credentials").WithHint(errorHint)
	}

	// Get the node auth config which contains OAuth2 settings
	nodeAuthConfig, err := api.Agent().NodeAuthConfig(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("an error has occurred fetching the node authentication details")
		return bacerrors.New("an error has occurred fetching the node authentication details").
			WithHint(errorHint)
	}

	// Check if OAuth2 is configured
	if nodeAuthConfig.Config.ProviderName == "" || nodeAuthConfig.Config.ProviderID == "" {
		log.Debug().Msg("orchestrator not configured with SSO login support. OAuth2 not configured on this server")
		return bacerrors.New("orchestrator does not support logging in using SSO").
			WithHint(errorHint)
	}

	// Create an OAuth2 service with the config from the server
	oauth2Service := sso.NewOAuth2Service(nodeAuthConfig.Config)

	// Step 1: Initiate the device code flow
	deviceCodeResp, err := oauth2Service.InitiateDeviceCodeFlow(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("failed to initiate oauth device code flow")
		return bacerrors.New("unable to initiate SSO login flow").
			WithHint(errorHint)
	}

	// Step 2: Show authentication instructions to the user
	printDeviceCodeInstructions(deviceCodeResp, nodeAuthConfig.Config.ProviderName, cmd.OutOrStdout())

	// Step 3: Create a timeout context for polling based on the expiry time
	timeoutCtx, cancel := context.WithTimeout(
		ctx,
		time.Duration(deviceCodeResp.ExpiresIn)*time.Second,
	)
	defer cancel()

	// Step 4: Start polling for the token
	token, err := oauth2Service.PollForToken(timeoutCtx, deviceCodeResp.DeviceCode)
	if err != nil {
		log.Debug().Err(err).Msg("error while polling for JWT token")
		return bacerrors.New("unable to finish SSO login flow").
			WithHint(errorHint)
	}

	// Get existing token if available
	persistableSSOCredentials := apimodels.HTTPCredential{
		Scheme: urlScheme,
		Value:  token.AccessToken,
	}

	err = util.WriteToken(authTokenPath, apiURL, &persistableSSOCredentials)
	if err != nil {
		log.Debug().Err(err).Msg("failed to write SSO JWTToken")
		return bacerrors.New("unable to save temporary SSO credentials").
			WithHint(errorHint)
	}

	// Save token to profile (non-fatal errors - don't break login flow)
	profilesDir := filepath.Join(cfg.DataDir, "profiles")
	store := profile.NewStore(profilesDir)
	profileName := endpointToProfileName(apiURL)
	loader := profile.NewLoader(store, "", "")

	p, err := loader.LoadOrCreate(profileName, apiURL)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to create/load profile for SSO token")
	} else {
		p.Auth = &profile.AuthConfig{Token: token.AccessToken}
		if err := store.Save(profileName, p); err != nil {
			log.Warn().Err(err).Msg("Failed to save SSO token to profile")
		} else {
			// Set as current if no current profile
			if current, _ := store.GetCurrent(); current == "" {
				_ = store.SetCurrent(profileName)
			}
			log.Debug().Str("profile", profileName).Msg("Saved SSO token to profile")
		}
	}

	fmt.Fprintf(os.Stderr, "\nSuccessfully authenticated with %s!\n", nodeAuthConfig.Config.ProviderName)

	return nil
}

// printDeviceCodeInstructions prints instructions for the user to complete the device code flow
func printDeviceCodeInstructions(deviceCode *sso.DeviceCodeResponse, providerName string, cmdOutput io.Writer) {
	fmt.Fprintln(cmdOutput, "")
	fmt.Fprintln(cmdOutput, "To login, please:")
	fmt.Fprintf(cmdOutput, "  1. Open this URL in your browser: %s\n", deviceCode.VerificationURI)
	fmt.Fprintf(cmdOutput, "  2. Enter this code: %s\n", deviceCode.UserCode)

	if deviceCode.VerificationURIComplete != "" {
		fmt.Fprintln(cmdOutput, "")
		fmt.Fprintln(cmdOutput, "Or, open this URL in your browser:")
		fmt.Fprintf(cmdOutput, "  %s\n", deviceCode.VerificationURIComplete)
		fmt.Fprintln(cmdOutput, "")
	}

	fmt.Fprintf(cmdOutput, "Waiting for authentication with %s... (press Ctrl+C to cancel)\n", providerName)
}

// endpointToProfileName converts an endpoint URL to a valid profile name.
// Example: "https://prod.example.com:443" -> "prod_example_com_443"
func endpointToProfileName(endpoint string) string {
	// Parse as URL to extract meaningful parts
	u, err := url.Parse(endpoint)
	if err != nil {
		// Fallback to simple sanitization
		return sanitizeProfileName(endpoint)
	}

	// Build profile name from host and port
	name := u.Hostname()
	if u.Port() != "" {
		name = name + "_" + u.Port()
	}

	return sanitizeProfileName(name)
}

// sanitizeProfileName converts a string to a valid profile name.
// Replaces invalid characters with underscores.
func sanitizeProfileName(name string) string {
	// Replace common URL separators and invalid characters
	name = strings.ReplaceAll(name, "://", "_")
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "-", "_")

	// Remove consecutive underscores
	re := regexp.MustCompile(`_+`)
	name = re.ReplaceAllString(name, "_")

	// Trim leading/trailing underscores
	name = strings.Trim(name, "_")

	// Ensure non-empty
	if name == "" {
		return "default"
	}

	return name
}

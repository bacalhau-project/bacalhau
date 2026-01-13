package util

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util/auth"
	"github.com/bacalhau-project/bacalhau/pkg/common"
	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

// ReadTokenFn is a function type for the ReadToken function that can be overridden for testing
var ReadTokenFn = ReadToken

type APIClientManager struct {
	cmd         *cobra.Command
	cfg         types.Bacalhau
	baseURL     string
	profile     *profile.Profile
	profileName string
}

func NewAPIClientManager(cmd *cobra.Command, cfg types.Bacalhau) *APIClientManager {
	cm := &APIClientManager{
		cmd: cmd,
		cfg: cfg,
	}

	// Load active profile from context
	profilesDir := filepath.Join(cfg.DataDir, "profiles")
	store := profile.NewStore(profilesDir)
	flagValue, envValue := GetProfileFromContext(cmd.Context())
	loader := profile.NewLoader(store, flagValue, envValue)

	p, name, err := loader.Load()
	if err != nil {
		log.Debug().Err(err).Msg("Failed to load profile")
	}

	if p != nil {
		// Use profile for connection
		cm.profile = p
		cm.profileName = name
		cm.baseURL = p.Endpoint
		log.Debug().Str("profile", name).Str("endpoint", p.Endpoint).Msg("Using profile for API connection")
	} else if apiEndpointExplicitlySet(cfg.API) {
		// Fall back to --api-host/--api-port flags if explicitly provided
		cm.baseURL, _ = ConstructAPIEndpoint(cfg.API)
		log.Debug().Str("endpoint", cm.baseURL).Msg("Using explicit API flags for connection")
	}
	// If neither profile nor explicit flags are set, baseURL will be empty
	// and client calls will fail with a clear error message.

	return cm
}

// apiEndpointExplicitlySet returns true if the API endpoint was explicitly set
// via flags or environment variables (not just default values from config).
func apiEndpointExplicitlySet(api types.API) bool {
	// Default values from pkg/config/defaults.go are Host="0.0.0.0" Port=1234
	// If either is different, explicit values were provided
	return api.Host != "0.0.0.0" || api.Port != 1234
}

var ErrNoProfile = fmt.Errorf("no profile configured. Create one with: bacalhau profile save <name> --endpoint <url>")

func (cm *APIClientManager) GetUnauthenticatedAPIClient() (clientv2.API, error) {
	if cm.baseURL == "" {
		return nil, ErrNoProfile
	}

	apiRequestOptions, err := cm.generateAPIRequestsOptions()
	if err != nil {
		return nil, err
	}

	return clientv2.New(cm.baseURL, apiRequestOptions...), nil
}

func (cm *APIClientManager) GetAuthenticatedAPIClient() (clientv2.API, error) {
	if cm.baseURL == "" {
		return nil, ErrNoProfile
	}

	apiRequestsOptions, err := cm.generateAPIRequestsOptions()
	if err != nil {
		return nil, err
	}

	apiAuthAPIKey, basicAuthUsername, basicAuthPassword := extractAuthCredentialsFromEnvVariables()
	var resolvedAuthToken *apimodels.HTTPCredential

	// Check if the credentials are valid
	apiKeyOrBasicAuthFlowEnabled, credentialScheme, credentialString, err := resolveAuthCredentials(
		apiAuthAPIKey,
		basicAuthUsername,
		basicAuthPassword,
	)
	if err != nil {
		return nil, fmt.Errorf("authentication error: %v", err)
	}

	// Priority for auth token:
	// 1. Environment variables (API key or basic auth)
	// 2. Profile auth token
	// 3. Legacy SSO tokens file
	// 4. Legacy auth tokens file

	if apiKeyOrBasicAuthFlowEnabled {
		resolvedAuthToken = &apimodels.HTTPCredential{
			Scheme: credentialScheme,
			Value:  credentialString,
		}
		log.Debug().Msg("Using API Key or Basic Auth authentication credentials")
	} else if cm.profile != nil && cm.profile.GetToken() != "" {
		// Use token from active profile
		resolvedAuthToken = &apimodels.HTTPCredential{
			Scheme: "Bearer",
			Value:  cm.profile.GetToken(),
		}
		log.Debug().Str("profile", cm.profileName).Msg("Using profile authentication token")
	} else {
		// Legacy fallback - try SSO tokens file, then legacy auth tokens
		ssoAuthTokenPath, err := cm.cfg.JWTTokensPath()
		if err != nil {
			log.Warn().Err(err).Msg("Failed to read access jwt tokens path")
		} else {
			existingSSOCredential, err := ReadTokenFn(ssoAuthTokenPath, cm.baseURL)
			if err != nil {
				log.Debug().Err(err).Msg("No SSO token found in legacy file")
			} else if existingSSOCredential != nil {
				resolvedAuthToken = existingSSOCredential
				log.Debug().Msg("Using legacy SSO authentication credentials")
			}
		}

		if resolvedAuthToken == nil {
			legacyAuthTokenFilePath, err := cm.cfg.AuthTokensPath()
			if err != nil {
				log.Warn().Err(err).Msg("Failed to read access tokens path")
			} else {
				resolvedAuthToken, err = ReadTokenFn(legacyAuthTokenFilePath, cm.baseURL)
				if err != nil {
					log.Debug().Err(err).Msg("No legacy auth token found")
				}
			}
		}
	}

	// Legacy Auth Flow for interactive authentication
	userKeyPath, err := cm.cfg.UserKeyPath()
	if err != nil {
		return nil, err
	}

	legacyAuthTokenFilePath, _ := cm.cfg.AuthTokensPath()
	newAuthenticationFlowEnabled := apiKeyOrBasicAuthFlowEnabled || (cm.profile != nil && cm.profile.GetToken() != "")

	return clientv2.NewAPI(
		&clientv2.AuthenticatingClient{
			Client:                       clientv2.NewHTTPClient(cm.baseURL, apiRequestsOptions...),
			Credential:                   resolvedAuthToken,
			NewAuthenticationFlowEnabled: newAuthenticationFlowEnabled,
			PersistCredential: func(cred *apimodels.HTTPCredential) error {
				return WriteToken(legacyAuthTokenFilePath, cm.baseURL, cred)
			},
			Authenticate: func(ctx context.Context, a *clientv2.Auth) (*apimodels.HTTPCredential, error) {
				return auth.RunAuthenticationFlow(ctx, cm.cmd, a, userKeyPath)
			},
		},
	), nil
}

// generateAPIRequestsOptions creates HTTP client options using profile or explicit flag TLS settings.
func (cm *APIClientManager) generateAPIRequestsOptions() ([]clientv2.OptionFn, error) {
	var useTLS, insecure bool
	var caFile string

	if cm.profile != nil {
		// Use profile TLS settings
		insecure = cm.profile.IsInsecure()
		useTLS = strings.HasPrefix(cm.baseURL, "https://")
	} else if apiEndpointExplicitlySet(cm.cfg.API) {
		// Use explicit flag TLS settings
		tlsCfg := cm.cfg.API.TLS
		useTLS = tlsCfg.UseTLS
		insecure = tlsCfg.Insecure
		caFile = tlsCfg.CAFile
	}
	// If neither, useTLS and insecure default to false, which is safe

	if caFile != "" {
		if _, err := os.Stat(caFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("CA certificate file %q does not exists", caFile)
		} else if err != nil {
			return nil, fmt.Errorf("CA certificate file %q cannot be read: %w", caFile, err)
		}
	}

	bv := version.Get()
	headers := map[string][]string{
		apimodels.HTTPHeaderBacalhauGitVersion: {bv.GitVersion},
		apimodels.HTTPHeaderBacalhauGitCommit:  {bv.GitCommit},
		apimodels.HTTPHeaderBacalhauBuildDate:  {bv.BuildDate.UTC().String()},
		apimodels.HTTPHeaderBacalhauBuildOS:    {bv.GOOS},
		apimodels.HTTPHeaderBacalhauArch:       {bv.GOARCH},
	}

	sysmeta, err := repo.LoadSystemMetadata(cm.cfg.DataDir)
	if err == nil {
		if sysmeta.InstanceID != "" {
			headers[apimodels.HTTPHeaderBacalhauInstanceID] = []string{sysmeta.InstanceID}
		}
	} else {
		log.Debug().Err(err).Msg("failed to load system metadata from repo path")
	}

	if installationID := system.InstallationID(); installationID != "" {
		headers[apimodels.HTTPHeaderBacalhauInstallationID] = []string{installationID}
	}

	opts := []clientv2.OptionFn{
		clientv2.WithCACertificate(caFile),
		clientv2.WithInsecureTLS(insecure),
		clientv2.WithTLS(useTLS),
		clientv2.WithHeaders(headers),
	}

	return opts, nil
}

func ConstructAPIEndpoint(apiCfg types.API) (string, string) {
	tlsCfg := apiCfg.TLS
	apiHost := apiCfg.Host
	apiPort := apiCfg.Port

	// set the client api host to localhost if it is 0.0.0.0
	if apiHost == "0.0.0.0" {
		apiHost = "127.0.0.1"
	}

	var baseURL string
	var scheme string

	if isValidURL, processedURL, detectedScheme := parseURL(apiHost, apiPort); isValidURL {
		baseURL = processedURL
		scheme = detectedScheme
	} else {
		scheme = "http"
		if tlsCfg.UseTLS {
			scheme = "https"
		}
		baseURL = fmt.Sprintf("%s://%s:%d", scheme, apiHost, apiPort)
	}

	return baseURL, scheme
}

func parseURL(rawURL string, defaultPort int) (bool, string, string) {
	// Remove any whitespace
	rawURL = strings.TrimSpace(rawURL)

	// Parse the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false, "", ""
	}

	// Check if the URL has a scheme and host
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return false, "", ""
	}

	// Check if scheme is http or https
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return false, "", ""
	}

	// Reject URLs with path, query, or fragment
	if parsedURL.Path != "" && parsedURL.Path != "/" {
		return false, "", ""
	}
	if parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return false, "", ""
	}

	// Handle port parsing for IPv4 and IPv6
	host := parsedURL.Host
	var port string
	processedHost, portStr, err := net.SplitHostPort(host)
	if err != nil {
		// No port specified in the URL
		// Clean up brackets if present for IPv6
		processedHost = strings.Trim(host, "[]")
		port = fmt.Sprintf("%d", defaultPort)
	} else {
		port = portStr
	}

	// Use net.JoinHostPort to properly handle IPv6 brackets
	hostPort := net.JoinHostPort(processedHost, port)
	finalURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, hostPort)

	return true, finalURL, parsedURL.Scheme
}

// resolveAuthCredentials processes authentication credentials and returns whether they should be used,
// along with the appropriate authentication scheme and credential value.
//
// Returns:
// - newAuthFlowEnabled: true if valid credentials were provided and should be used
// - authScheme: the authentication scheme ("Bearer" or "Basic")
// - credentialValue: the credential value (API key or Base64-encoded username:password)
// - error: validation error, if any
func resolveAuthCredentials(
	apiKey,
	basicAuthUsername,
	basicAuthPassword string,
) (
	newAuthFlowEnabled bool,
	authScheme string,
	credentialValue string,
	err error,
) {
	// Trim spaces from all credentials
	apiKey = strings.TrimSpace(apiKey)
	basicAuthUsername = strings.TrimSpace(basicAuthUsername)
	basicAuthPassword = strings.TrimSpace(basicAuthPassword)

	// Check if any credentials are provided
	anyCredentialsProvided := apiKey != "" || basicAuthUsername != "" || basicAuthPassword != ""

	if !anyCredentialsProvided {
		// No credentials provided, use legacy auth flow
		return false, "", "", nil
	}

	// At this point, we know some credentials were provided, so we'll use the new auth flow
	newAuthFlowEnabled = true

	// Check if trying to use both authentication methods
	hasAPIKey := apiKey != ""
	hasBasicAuthUsername := basicAuthUsername != ""
	hasBasicAuthPassword := basicAuthPassword != ""

	// Error if mixing authentication types
	if hasAPIKey && (hasBasicAuthUsername || hasBasicAuthPassword) {
		return newAuthFlowEnabled, "", "", fmt.Errorf("can't use both %s and %s/%s simultaneously",
			common.BacalhauAPIKey, common.BacalhauAPIUsername, common.BacalhauAPIPassword)
	}

	// Handle API key authentication
	if hasAPIKey {
		return newAuthFlowEnabled, "Bearer", apiKey, nil
	}

	// Handle username/password basic authentication
	if hasBasicAuthUsername && hasBasicAuthPassword {
		// Format basic auth credentials (username:password) as Base64. RFC dictated
		basicAuthString := fmt.Sprintf("%s:%s", basicAuthUsername, basicAuthPassword)
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(basicAuthString))
		return newAuthFlowEnabled, "Basic", encodedAuth, nil
	}

	// Handle incomplete basic auth credentials
	if hasBasicAuthUsername {
		return newAuthFlowEnabled, "", "", fmt.Errorf("%s provided but not %s",
			common.BacalhauAPIUsername, common.BacalhauAPIPassword)
	}
	if hasBasicAuthPassword {
		return newAuthFlowEnabled, "", "", fmt.Errorf("%s provided but not %s",
			common.BacalhauAPIPassword, common.BacalhauAPIUsername)
	}

	// This should never happen given the checks above
	return newAuthFlowEnabled, "", "", fmt.Errorf("unable to decide authentication method")
}

func extractAuthCredentialsFromEnvVariables() (string, string, string) {
	apiKey := strings.TrimSpace(os.Getenv(common.BacalhauAPIKey))
	basicAuthUsername := strings.TrimSpace(os.Getenv(common.BacalhauAPIUsername))
	basicAuthPassword := strings.TrimSpace(os.Getenv(common.BacalhauAPIPassword))

	return apiKey, basicAuthUsername, basicAuthPassword
}

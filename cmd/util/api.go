package util

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util/auth"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

// ReadTokenFn is a function type for the ReadToken function that can be overridden for testing
var ReadTokenFn = ReadToken

//nolint:funlen
func GetAPIClientV2(cmd *cobra.Command, cfg types.Bacalhau) (clientv2.API, error) {
	apiAuthAPIKey, basicAuthUsername, basicAuthPassword := extractAuthCredentialsFromEnvVariables()
	baseURL, _ := ConstructAPIEndpoint(cfg.API)
	tlsCfg := cfg.API.TLS

	if tlsCfg.CAFile != "" {
		if _, err := os.Stat(tlsCfg.CAFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("CA certificate file %q does not exists", tlsCfg.CAFile)
		} else if err != nil {
			return nil, fmt.Errorf("CA certificate file %q cannot be read: %w", tlsCfg.CAFile, err)
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

	sysmeta, err := repo.LoadSystemMetadata(cfg.DataDir)
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
		clientv2.WithCACertificate(tlsCfg.CAFile),
		clientv2.WithInsecureTLS(tlsCfg.Insecure),
		clientv2.WithTLS(tlsCfg.UseTLS),
		clientv2.WithHeaders(headers),
	}

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

	legacyAuthTokenFilePath, err := cfg.AuthTokensPath()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to read access tokens path – API calls will be without authorization")
	}

	// Try to get the tokens file for SSO tokens
	ssoAuthTokenPath, err := cfg.JWTTokensPath()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to read access jwt tokens path")
	}

	// Do not error out if we are not able to do that , just log
	existingSSOCredential, err := ReadTokenFn(ssoAuthTokenPath, baseURL)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to read SSO access tokens file")
	}

	// If credentials are provided, add them to the headers
	if apiKeyOrBasicAuthFlowEnabled {
		resolvedAuthToken = &apimodels.HTTPCredential{
			Scheme: credentialScheme,
			Value:  credentialString,
		}
		log.Debug().Msg("Using API Key or Basic Auth authentication credentials")
	} else if existingSSOCredential != nil {
		resolvedAuthToken = existingSSOCredential
		log.Debug().Msg("Using SSO authentication credentials")
	} else {
		resolvedAuthToken, err = ReadTokenFn(legacyAuthTokenFilePath, baseURL)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to read access tokens – API calls will be without authorization")
		}
	}

	newAuthenticationFlowEnabled := apiKeyOrBasicAuthFlowEnabled || existingSSOCredential != nil
	skipAuthentication := cmd.Use == "sso" || cmd.Use == "version" || cmd.Use == "alive"

	userKeyPath, err := cfg.UserKeyPath()
	if err != nil {
		return nil, err
	}

	return clientv2.NewAPI(
		&clientv2.AuthenticatingClient{
			Client:                       clientv2.NewHTTPClient(baseURL, opts...),
			Credential:                   resolvedAuthToken,
			NewAuthenticationFlowEnabled: newAuthenticationFlowEnabled,
			SkipAuthentication:           skipAuthentication,
			PersistCredential: func(cred *apimodels.HTTPCredential) error {
				return WriteToken(legacyAuthTokenFilePath, baseURL, cred)
			},
			Authenticate: func(ctx context.Context, a *clientv2.Auth) (*apimodels.HTTPCredential, error) {
				return auth.RunAuthenticationFlow(ctx, cmd, a, userKeyPath)
			},
		},
	), nil
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
		return newAuthFlowEnabled, "", "", fmt.Errorf("can't use both " +
			"BACALHAU_API_KEY and BACALHAU_API_USERNAME/BACALHAU_API_PASSWORD simultaneously")
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
		return newAuthFlowEnabled, "", "", fmt.Errorf("BACALHAU_API_USERNAME provided but not BACALHAU_API_PASSWORD")
	}
	if hasBasicAuthPassword {
		return newAuthFlowEnabled, "", "", fmt.Errorf("BACALHAU_API_PASSWORD provided but not BACALHAU_API_USERNAME")
	}

	// This should never happen given the checks above
	return newAuthFlowEnabled, "", "", fmt.Errorf("unable to decide authentication method")
}

func extractAuthCredentialsFromEnvVariables() (string, string, string) {
	apiKey := strings.TrimSpace(os.Getenv("BACALHAU_API_KEY"))
	basicAuthUsername := strings.TrimSpace(os.Getenv("BACALHAU_API_USERNAME"))
	basicAuthPassword := strings.TrimSpace(os.Getenv("BACALHAU_API_PASSWORD"))

	return apiKey, basicAuthUsername, basicAuthPassword
}

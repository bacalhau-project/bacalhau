package util

import (
	"context"
	"fmt"
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

func GetAPIClientV2(cmd *cobra.Command, cfg types.Bacalhau) (clientv2.API, error) {
	tlsCfg := cfg.API.TLS
	apiHost := cfg.API.Host
	apiPort := cfg.API.Port

	// set the client api host to localhost if it is 0.0.0.0
	if apiHost == "0.0.0.0" {
		apiHost = "127.0.0.1"
	}

	if tlsCfg.CAFile != "" {
		if _, err := os.Stat(tlsCfg.CAFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("CA certificate file %q does not exists", tlsCfg.CAFile)
		} else if err != nil {
			return nil, fmt.Errorf("CA certificate file %q cannot be read: %w", tlsCfg.CAFile, err)
		}
	}

	var base string

	if isValidUrl, processedUrl := parseURL(apiHost, apiPort); isValidUrl {
		base = processedUrl
	} else {
		scheme := "http"
		if tlsCfg.UseTLS {
			scheme = "https"
		}
		base = fmt.Sprintf("%s://%s:%d", scheme, apiHost, apiPort)
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

	authTokenPath, err := cfg.AuthTokensPath()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to read access tokens path – API calls will be without authorization")
	}
	existingAuthToken, err := ReadToken(authTokenPath, base)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to read access tokens – API calls will be without authorization")
	}

	userKeyPath, err := cfg.UserKeyPath()
	if err != nil {
		return nil, err
	}

	return clientv2.NewAPI(
		&clientv2.AuthenticatingClient{
			Client:     clientv2.NewHTTPClient(base, opts...),
			Credential: existingAuthToken,
			PersistCredential: func(cred *apimodels.HTTPCredential) error {
				return WriteToken(authTokenPath, base, cred)
			},
			Authenticate: func(ctx context.Context, a *clientv2.Auth) (*apimodels.HTTPCredential, error) {
				return auth.RunAuthenticationFlow(ctx, cmd, a, userKeyPath)
			},
		},
	), nil
}

func parseURL(rawURL string, defaultPort int) (bool, string) {
	// Remove any whitespace
	rawURL = strings.TrimSpace(rawURL)

	// Parse the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false, ""
	}

	// Check if the URL has a scheme and host
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return false, ""
	}

	// Check if scheme is http or https
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return false, ""
	}

	// Reject URLs with path, query, or fragment
	if parsedURL.Path != "" && parsedURL.Path != "/" {
		return false, ""
	}
	if parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return false, ""
	}

	// Extract host and port
	host := parsedURL.Host
	var port string
	var processedHost string

	// Handle IPv6 addresses with port
	if strings.HasPrefix(host, "[") && strings.Contains(host, "]:") {
		// IPv6 with port
		lastColon := strings.LastIndex(host, ":")
		port = host[lastColon+1:]
		processedHost = host[:lastColon] // Keep the brackets
	} else if strings.HasPrefix(host, "[") {
		// IPv6 without port
		processedHost = host
		port = fmt.Sprintf("%d", defaultPort)
	} else if strings.Count(host, ":") == 1 {
		// IPv4 or hostname with port
		hostParts := strings.Split(host, ":")
		processedHost = hostParts[0]
		port = hostParts[1]
	} else {
		// IPv4 or hostname without port
		processedHost = host
		port = fmt.Sprintf("%d", defaultPort)
	}

	// Construct final URL with just scheme, host, and port
	finalURL := parsedURL.Scheme + "://"

	// If it's an IPv6 address without brackets, add them
	if strings.Count(processedHost, ":") > 1 && !strings.HasPrefix(processedHost, "[") {
		finalURL += "[" + processedHost + "]"
	} else {
		finalURL += processedHost
	}

	finalURL += ":" + port

	return true, finalURL
}

package util

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util/auth"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

func GetAPIClient(cfg types.BacalhauConfig) (*client.APIClient, error) {
	tlsCfg := cfg.Node.ClientAPI.ClientTLS
	apiHost := cfg.Node.ClientAPI.Host
	apiPort := cfg.Node.ClientAPI.Port
	tokenPath := cfg.Auth.TokensPath

	if tlsCfg.CACert != "" {
		if _, err := os.Stat(tlsCfg.CACert); os.IsNotExist(err) {
			return nil, fmt.Errorf("CA certificate file %q does not exists", tlsCfg.CACert)
		} else if err != nil {
			return nil, fmt.Errorf("CA certificate file %q cannot be read: %w", tlsCfg.CACert, err)
		}
	}

	apiClient, err := client.NewAPIClient(client.LegacyTLSSupport(tlsCfg), cfg.User, apiHost, uint16(apiPort))
	if err != nil {
		return nil, err
	}

	apiSheme := "http"
	if tlsCfg.UseTLS {
		apiSheme = "https"
	}

	if token, err := ReadToken(tokenPath, fmt.Sprintf("%s://%s:%d", apiSheme, apiHost, apiPort)); err != nil {
		log.Warn().Err(err).Msg("Failed to read access tokens – API calls will be without authorization")
	} else if token != nil {
		apiClient.DefaultHeaders["Authorization"] = token.String()
	}

	return apiClient, nil
}

func GetAPIClientV2(cmd *cobra.Command, cfg types.BacalhauConfig) (clientv2.API, error) {
	tlsCfg := cfg.Node.ClientAPI.ClientTLS
	apiHost := cfg.Node.ClientAPI.Host
	apiPort := cfg.Node.ClientAPI.Port
	tokenPath := cfg.Auth.TokensPath
	clientKeyPath := cfg.User.KeyPath

	if tlsCfg.CACert != "" {
		if _, err := os.Stat(tlsCfg.CACert); os.IsNotExist(err) {
			return nil, fmt.Errorf("CA certificate file %q does not exists", tlsCfg.CACert)
		} else if err != nil {
			return nil, fmt.Errorf("CA certificate file %q cannot be read: %w", tlsCfg.CACert, err)
		}
	}

	apiSheme := "http"
	if tlsCfg.UseTLS {
		apiSheme = "https"
	}
	base := fmt.Sprintf("%s://%s:%d", apiSheme, apiHost, apiPort)

	bv := version.Get()
	headers := map[string][]string{
		apimodels.HTTPHeaderBacalhauGitVersion: {bv.GitVersion},
		apimodels.HTTPHeaderBacalhauGitCommit:  {bv.GitCommit},
		apimodels.HTTPHeaderBacalhauBuildDate:  {bv.BuildDate.UTC().String()},
		apimodels.HTTPHeaderBacalhauBuildOS:    {bv.GOOS},
		apimodels.HTTPHeaderBacalhauArch:       {bv.GOARCH},
	}

	opts := []clientv2.OptionFn{
		clientv2.WithCACertificate(tlsCfg.CACert),
		clientv2.WithInsecureTLS(tlsCfg.Insecure),
		clientv2.WithTLS(tlsCfg.UseTLS),
		clientv2.WithHeaders(headers),
	}

	existingAuthToken, err := ReadToken(tokenPath, base)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to read access tokens – API calls will be without authorization")
	}

	return clientv2.NewAPI(
		&clientv2.AuthenticatingClient{
			Client:     clientv2.NewHTTPClient(base, opts...),
			Credential: existingAuthToken,
			PersistCredential: func(cred *apimodels.HTTPCredential) error {
				return WriteToken(tokenPath, base, cred)
			},
			Authenticate: func(ctx context.Context, a *clientv2.Auth) (*apimodels.HTTPCredential, error) {
				return auth.RunAuthenticationFlow(ctx, cmd, a, clientKeyPath)
			},
		},
	), nil
}

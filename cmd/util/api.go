package util

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util/auth"
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

func GetAPIClientV2(cmd *cobra.Command, cfg types2.Bacalhau) (clientv2.API, error) {
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

	opts := []clientv2.OptionFn{
		clientv2.WithCACertificate(tlsCfg.CAFile),
		//clientv2.WithInsecureTLS(tlsCfg.Insecure),
		//clientv2.WithTLS(tlsCfg.UseTLS),
		clientv2.WithHeaders(headers),
	}

	authTokenPath, err := cfg.AuthTokensPath()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to read access tokens path – API calls will be without authorization")
	}
	existingAuthToken, err := ReadToken(authTokenPath, cfg.API.Address)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to read access tokens – API calls will be without authorization")
	}

	userKeyPath, err := cfg.UserKeyPath()
	if err != nil {
		return nil, err
	}

	parsedURL, err := url.Parse(cfg.API.Address)
	if err != nil {
		return nil, err
	}
	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "http"
	}

	base := parsedURL.String()
	return clientv2.NewAPI(
		&clientv2.AuthenticatingClient{
			Client:     clientv2.NewHTTPClient(base, opts...),
			Credential: existingAuthToken,
			PersistCredential: func(cred *apimodels.HTTPCredential) error {
				return WriteToken(authTokenPath, cfg.API.Address, cred)
			},
			Authenticate: func(ctx context.Context, a *clientv2.Auth) (*apimodels.HTTPCredential, error) {
				return auth.RunAuthenticationFlow(ctx, cmd, a, userKeyPath)
			},
		},
	), nil
}

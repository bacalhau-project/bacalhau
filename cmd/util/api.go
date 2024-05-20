package util

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util/auth"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

func GetAPIClient(ctx context.Context) *client.APIClient {
	legacyTLS := client.LegacyTLSSupport(config.ClientTLSConfig())
	apiClient := client.NewAPIClient(legacyTLS, config.ClientAPIHost(), config.ClientAPIPort())

	if token, err := ReadToken(config.ClientAPIBase()); err != nil {
		log.Warn().Err(err).Msg("Failed to read access tokens – API calls will be without authorization")
	} else if token != nil {
		apiClient.DefaultHeaders["Authorization"] = token.String()
	}

	return apiClient
}

func GetAPIClientV2(cmd *cobra.Command) clientv2.API {
	base := config.ClientAPIBase()
	tlsConfig := config.ClientTLSConfig()

	bv := version.Get()
	headers := map[string][]string{
		apimodels.HTTPHeaderBacalhauGitVersion: {bv.GitVersion},
		apimodels.HTTPHeaderBacalhauGitCommit:  {bv.GitCommit},
		apimodels.HTTPHeaderBacalhauBuildDate:  {bv.BuildDate.UTC().String()},
		apimodels.HTTPHeaderBacalhauBuildOS:    {bv.GOOS},
		apimodels.HTTPHeaderBacalhauArch:       {bv.GOARCH},
	}

	opts := []clientv2.OptionFn{
		clientv2.WithCACertificate(tlsConfig.CACert),
		clientv2.WithInsecureTLS(tlsConfig.Insecure),
		clientv2.WithTLS(tlsConfig.UseTLS),
		clientv2.WithHeaders(headers),
	}

	existingAuthToken, err := ReadToken(base)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to read access tokens – API calls will be without authorization")
	}

	return clientv2.NewAPI(
		&clientv2.AuthenticatingClient{
			Client:     clientv2.NewHTTPClient(base, opts...),
			Credential: existingAuthToken,
			PersistCredential: func(cred *apimodels.HTTPCredential) error {
				return WriteToken(base, cred)
			},
			Authenticate: func(ctx context.Context, a *clientv2.Auth) (*apimodels.HTTPCredential, error) {
				return auth.RunAuthenticationFlow(ctx, cmd, a)
			},
		},
	)
}

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

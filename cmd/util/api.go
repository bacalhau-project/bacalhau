package util

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util/auth"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

func GetAPIClient(c *config.Config) (*client.APIClient, error) {
	repoDir, err := c.RepoPath()
	if err != nil {
		return nil, err
	}
	// TODO(forrest) [refactor]: a repo is required here as it contains the clients
	// private key and a pk is needed in order to perform the authentication flow
	// iff the configured authentication mode is challenge. We could decide to only
	// require a repo here if the client _wants_ to communicate with a requester
	// that uses authorization. In the event a requester doesn't need authorization
	// we could use a un-authenticated client and not create a repo here.
	_, err = setup.SetupBacalhauRepo(repoDir, c)
	if err != nil {
		return nil, err
	}

	var tlsCfg types.ClientTLSConfig
	if err := c.ForKey(types.NodeClientAPITLS, &tlsCfg); err != nil {
		panic(err)
	}
	var apiHost string
	if err := c.ForKey(types.NodeClientAPIHost, &apiHost); err != nil {
		panic(err)
	}
	var apiPort uint16
	if err := c.ForKey(types.NodeClientAPIPort, &apiPort); err != nil {
		panic(err)
	}
	legacyTLS := client.LegacyTLSSupport(tlsCfg)
	apiClient := client.NewAPIClient(legacyTLS, apiHost, apiPort)

	apiSheme := "http"
	if tlsCfg.UseTLS {
		apiSheme = "https"
	}

	if token, err := ReadToken(c, fmt.Sprintf("%s://%s:%d", apiSheme, apiHost, apiPort)); err != nil {
		log.Warn().Err(err).Msg("Failed to read access tokens – API calls will be without authorization")
	} else if token != nil {
		apiClient.DefaultHeaders["Authorization"] = token.String()
	}

	return apiClient, nil
}

func GetAPIClientV2(cmd *cobra.Command, c *config.Config) (clientv2.API, error) {
	repoDir, err := c.RepoPath()
	if err != nil {
		return nil, err
	}
	// TODO(forrest) [refactor]: a repo is required here as it contains the clients
	// private key and a pk is needed in order to perform the authentication flow
	// iff the configured authentication mode is challenge. We could decide to only
	// require a repo here if the client _wants_ to communicate with a requester
	// that uses authorization. In the event a requester doesn't need authorization
	// we could use a un-authenticated client and not create a repo here.
	fsr, err := setup.SetupBacalhauRepo(repoDir, c)
	if err != nil {
		return nil, err
	}

	var tlsCfg types.ClientTLSConfig
	if err := c.ForKey(types.NodeClientAPITLS, &tlsCfg); err != nil {
		return nil, err
	}
	var apiHost string
	if err := c.ForKey(types.NodeClientAPIHost, &apiHost); err != nil {
		return nil, err
	}
	var apiPort uint16
	if err := c.ForKey(types.NodeClientAPIPort, &apiPort); err != nil {
		return nil, err
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

	existingAuthToken, err := ReadToken(c, base)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to read access tokens – API calls will be without authorization")
	}

	return clientv2.NewAPI(
		&clientv2.AuthenticatingClient{
			Client:     clientv2.NewHTTPClient(base, opts...),
			Credential: existingAuthToken,
			PersistCredential: func(cred *apimodels.HTTPCredential) error {
				return WriteToken(base, c, cred)
			},
			Authenticate: func(a *clientv2.Auth) (*apimodels.HTTPCredential, error) {
				return auth.RunAuthenticationFlow(cmd, a, fsr)
			},
		},
	), nil
}

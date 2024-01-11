package util

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

func GetAPIClient(ctx context.Context) *client.APIClient {
	legacyTLS := client.LegacyTLSSupport(config.ClientTLSConfig())
	return client.NewAPIClient(legacyTLS, config.ClientAPIHost(), config.ClientAPIPort())
}

func GetAPIClientV2(ctx context.Context) *clientv2.Client {
	tlsConfig := config.ClientTLSConfig()

	scheme := "http"
	if tlsConfig.UseTLS {
		scheme = "https"
	}

	bv := version.Get()
	headers := map[string][]string{
		apimodels.HTTPHeaderClientMajorVersion: {bv.Major},
		apimodels.HTTPHeaderClientMinorVersion: {bv.Minor},
		apimodels.HTTPHeaderClientGitVersion:   {bv.GitVersion},
		apimodels.HTTPHeaderClientGitCommit:    {bv.GitCommit},
		apimodels.HTTPHeaderClientBuildDate:    {bv.BuildDate.UTC().String()},
		apimodels.HTTPHeaderClientBuildOS:      {bv.GOOS},
		apimodels.HTTPHeaderClientArch:         {bv.GOARCH},
	}
	return clientv2.New(clientv2.Options{
		Context: ctx,
		Address: fmt.Sprintf("%s://%s:%d", scheme, config.ClientAPIHost(), config.ClientAPIPort()),
	},
		clientv2.WithCACertificate(tlsConfig.CACert),
		clientv2.WithInsecureTLS(tlsConfig.Insecure),
		clientv2.WithTLS(tlsConfig.UseTLS),
		clientv2.WithHeaders(headers),
	)
}

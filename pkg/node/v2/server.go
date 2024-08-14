package v2

import (
	"crypto/rsa"

	"github.com/labstack/echo/v4"

	"github.com/bacalhau-project/bacalhau/pkg/authz"
	v2 "github.com/bacalhau-project/bacalhau/pkg/config/types/v2"
	"github.com/bacalhau-project/bacalhau/pkg/lib/policy"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

func SetupAPIServer(signingKey *rsa.PublicKey, cfg v2.Bacalhau) (*publicapi.Server, error) {
	authzPolicy, err := policy.FromPathOrDefault(cfg.Server.Auth.AccessPolicyPath, authz.AlwaysAllowPolicy)
	if err != nil {
		return nil, err
	}

	serverVersion := version.Get()
	serverParams := publicapi.ServerParams{
		Router:  echo.New(),
		Address: cfg.Server.Address,
		Port:    uint16(cfg.Server.Port),
		HostID:  cfg.Name,
		// NB(forrest) [breadcrumb] this used to happen in pkg/node.prepareConfig
		Config: publicapi.DefaultConfig(),

		Authorizer: authz.NewPolicyAuthorizer(authzPolicy, signingKey, cfg.Name),
		Headers: map[string]string{
			apimodels.HTTPHeaderBacalhauGitVersion: serverVersion.GitVersion,
			apimodels.HTTPHeaderBacalhauGitCommit:  serverVersion.GitCommit,
			apimodels.HTTPHeaderBacalhauBuildDate:  serverVersion.BuildDate.UTC().String(),
			apimodels.HTTPHeaderBacalhauBuildOS:    serverVersion.GOOS,
			apimodels.HTTPHeaderBacalhauArch:       serverVersion.GOARCH,
		},
	}

	// Only allow autocert for requester nodes
	if cfg.Orchestrator.Enabled {
		serverParams.AutoCertDomain = cfg.Server.TLS.AutoCert
		serverParams.AutoCertCache = cfg.Server.TLS.AutoCertCachePath
		serverParams.TLSCertificateFile = cfg.Server.TLS.Certificate
		serverParams.TLSKeyFile = cfg.Server.TLS.Key
	}

	apiServer, err := publicapi.NewAPIServer(serverParams)
	if err != nil {
		return nil, err
	}
	return apiServer, nil
}

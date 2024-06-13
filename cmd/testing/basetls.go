package cmdtesting

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	"github.com/bacalhau-project/bacalhau/pkg/test/teststack"
	"github.com/bacalhau-project/bacalhau/pkg/test/utils/certificates"
)

type BaseTLSSuite struct {
	BaseSuite
	TempCACertFilePath string
}

// before each test
func (s *BaseTLSSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())

	ctx := context.Background()
	requesterConfig, err := node.NewRequesterConfigWith(
		node.RequesterConfigParams{
			HousekeepingBackgroundTaskInterval: 1 * time.Second,
		},
	)
	s.Require().NoError(err)

	tempDir := s.T().TempDir()
	caCertPath := filepath.Join(tempDir, "ca_certificate.pem")
	caKeyPath := filepath.Join(tempDir, "ca_private_key.pem")
	serverCertPath := filepath.Join(tempDir, "server_certificate.pem")
	serverKeyPath := filepath.Join(tempDir, "server_private_key.pem")

	s.TempCACertFilePath = caCertPath

	// generate certificates
	caCert, err := certificates.NewTestCACertificate(caCertPath, caKeyPath)
	s.Require().NoError(err)
	_, err = caCert.CreateTestSignedCertificate(serverCertPath, serverKeyPath)
	s.Require().NoError(err)

	fsr, cfg := setup.SetupBacalhauRepoForTesting(s.T())

	computeConfig, err := node.NewComputeConfigWith(cfg.Node.ComputeStoragePath, node.ComputeConfigParams{
		JobSelectionPolicy: node.JobSelectionPolicy{
			Locality: semantic.Anywhere,
		},
	})
	s.Require().NoError(err)

	stack := teststack.Setup(ctx, s.T(),
		fsr, cfg,
		devstack.WithNumberOfHybridNodes(1),
		devstack.WithComputeConfig(computeConfig),
		devstack.WithRequesterConfig(requesterConfig),
		devstack.WithSelfSignedCertificate(serverCertPath, serverKeyPath),
		teststack.WithNoopExecutor(noop_executor.ExecutorConfig{}, cfg.Node.Compute.ManifestCache),
	)
	s.Node = stack.Nodes[0]
	s.Host = s.Node.APIServer.Address // NOTE: 0.0.0.0 will not work because we're testing TLS validation
	s.Port = s.Node.APIServer.Port
	s.Client, err = client.NewAPIClient(client.LegacyTLSSupport{UseTLS: true, Insecure: false}, cfg.User, s.Host, s.Port)
	s.Require().NoError(err)
	s.ClientV2 = clientv2.New(fmt.Sprintf("http://%s:%d", s.Host, s.Port), clientv2.WithTLS(true), clientv2.WithInsecureTLS(true))
}

// After each test
func (s *BaseTLSSuite) TearDownTest() {
	if s.Node != nil {
		s.Node.CleanupManager.Cleanup(context.Background())
	}
}

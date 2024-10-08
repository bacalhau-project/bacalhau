package cmdtesting

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
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

	cfg, err := config.NewTestConfig()
	s.Require().NoError(err)

	stack := teststack.Setup(ctx, s.T(),
		devstack.WithNumberOfHybridNodes(1),
		devstack.WithSelfSignedCertificate(serverCertPath, serverKeyPath),
		teststack.WithNoopExecutor(noop_executor.ExecutorConfig{}, cfg.Engines),
	)
	s.Node = stack.Nodes[0]
	s.Host = s.Node.APIServer.Address // NOTE: 0.0.0.0 will not work because we're testing TLS validation
	s.Port = s.Node.APIServer.Port
	s.ClientV2 = clientv2.New(fmt.Sprintf("http://%s:%d", s.Host, s.Port), clientv2.WithTLS(true), clientv2.WithInsecureTLS(true))
}

// After each test
func (s *BaseTLSSuite) TearDownTest() {
	if s.Node != nil {
		s.Node.CleanupManager.Cleanup(context.Background())
	}
}

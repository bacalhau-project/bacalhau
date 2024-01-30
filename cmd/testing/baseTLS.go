package cmdtesting

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/test/teststack"
)

type BaseTLSSuite struct {
	BaseSuite
}

// before each test
func (s *BaseTLSSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	util.Fatal = util.FakeFatalErrorHandler

	computeConfig, err := node.NewComputeConfigWith(node.ComputeConfigParams{
		JobSelectionPolicy: node.JobSelectionPolicy{
			Locality: semantic.Anywhere,
		},
	})
	s.Require().NoError(err)
	ctx := context.Background()
	requesterConfig, err := node.NewRequesterConfigWith(
		node.RequesterConfigParams{
			HousekeepingBackgroundTaskInterval: 1 * time.Second,
		},
	)
	s.Require().NoError(err)

	const serverCertPath = "../../testdata/certs/dev-server.crt"
	const serverKeyPath = "../../testdata/certs/dev-server.key"

	stack := teststack.Setup(ctx, s.T(),
		devstack.WithNumberOfHybridNodes(1),
		devstack.WithComputeConfig(computeConfig),
		devstack.WithRequesterConfig(requesterConfig),
		devstack.WithSelfSignedCertificate(serverCertPath, serverKeyPath),
		teststack.WithNoopExecutor(noop_executor.ExecutorConfig{}),
	)
	s.Node = stack.Nodes[0]
	s.Host = "127.0.0.1" //0.0.0.0 will not work because we're testing TLS validation
	s.Port = s.Node.APIServer.Port
	s.Client = client.NewAPIClient(client.LegacyTLSSupport{UseTLS: true, Insecure: false}, s.Host, s.Port)
	s.ClientV2 = clientv2.New(fmt.Sprintf("http://%s:%d", s.Host, s.Port), clientv2.WithTLS(true))
}

// After each test
func (s *BaseTLSSuite) TearDownTest() {
	util.Fatal = util.FakeFatalErrorHandler
	if s.Node != nil {
		s.Node.CleanupManager.Cleanup(context.Background())
	}
}

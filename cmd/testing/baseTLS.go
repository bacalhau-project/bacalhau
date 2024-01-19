package cmdtesting

import (
	"context"
	"fmt"
	"time"

	"github.com/stretchr/testify/suite"

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
	suite.Suite
	Node     *node.Node
	Client   *client.APIClient
	ClientV2 *clientv2.Client
	Host     string
	Port     uint16
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
	fmt.Printf("OLGIBBONS DEBUG: computeconfig: %#v\n", computeConfig)
	s.Require().NoError(err)
	ctx := context.Background()
	requesterConfig, err := node.NewRequesterConfigWith(
		node.RequesterConfigParams{
			HousekeepingBackgroundTaskInterval: 1 * time.Second,
		},
	)
	fmt.Printf("OLGIBBONS DEBUG: requesterconfig: %#v\n", requesterConfig)
	s.Require().NoError(err)

	stack := teststack.Setup(ctx, s.T(),
		devstack.WithNumberOfHybridNodes(1),
		devstack.WithComputeConfig(computeConfig),
		devstack.WithRequesterConfig(requesterConfig),
		//devstack.WithSelfSignedCertificate("/home/gibbons/Bacalhau/bacalhau/testdata/certs/dev-server.crt", "/home/gibbons/Bacalhau/bacalhau/testdata/certs/dev-server.key"), //olgibbons change this
		teststack.WithNoopExecutor(noop_executor.ExecutorConfig{}),
	)
	s.Node = stack.Nodes[0]
	s.Host = s.Node.APIServer.Address
	s.Port = s.Node.APIServer.Port
	fmt.Printf("OLGIBBONS DEBUG: host: %#v, port: %#v\n", s.Host, fmt.Sprint(s.Port))
	s.Client = client.NewAPIClient(client.LegacyTLSSupport{UseTLS: false, Insecure: false}, s.Host, s.Port)
	s.ClientV2 = clientv2.New(clientv2.Options{
		Address: fmt.Sprintf("http://%s:%d", s.Host, s.Port),
		TLS:     clientv2.TLSConfig{UseTLS: false, Insecure: false},
	})
}

// After each test
func (s *BaseTLSSuite) TearDownTest() {
	util.Fatal = util.FakeFatalErrorHandler
	if s.Node != nil {
		s.Node.CleanupManager.Cleanup(context.Background())
	}
}

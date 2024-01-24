package cmdtesting

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
	"golang.org/x/exp/slices"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"

	"github.com/bacalhau-project/bacalhau/cmd/cli"
	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/test/teststack"
)

type BaseSuite struct {
	suite.Suite
	Node     *node.Node
	Client   *client.APIClient
	ClientV2 *clientv2.Client
	Host     string
	Port     uint16
}

// before each test
func (s *BaseSuite) SetupTest() {
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
	stack := teststack.Setup(ctx, s.T(),
		devstack.WithNumberOfHybridNodes(1),
		devstack.WithComputeConfig(computeConfig),
		devstack.WithRequesterConfig(requesterConfig),
		teststack.WithNoopExecutor(noop_executor.ExecutorConfig{}),
	)
	s.Node = stack.Nodes[0]
	s.Host = s.Node.APIServer.Address
	s.Port = s.Node.APIServer.Port
	s.Client = client.NewAPIClient(client.NoTLS, s.Host, s.Port)
	s.ClientV2 = clientv2.New(clientv2.Options{
		Address: fmt.Sprintf("http://%s:%d", s.Host, s.Port),
	})
}

// After each test
func (s *BaseSuite) TearDownTest() {
	util.Fatal = util.FakeFatalErrorHandler
	if s.Node != nil {
		s.Node.CleanupManager.Cleanup(context.Background())
	}
}

// ExecuteTestCobraCommand executes a cobra command with the given arguments. The api-host and api-port
// flags are automatically added if they are not provided in `args`. They are set to the values of
// `s.Host` and `s.Port` respectively.
func (s *BaseSuite) ExecuteTestCobraCommand(args ...string) (c *cobra.Command, output string, err error) {
	return s.ExecuteTestCobraCommandWithStdin(nil, args...)
}

// ExecuteTestCobraCommandWithStdin executes a cobra command with the given arguments and with a specific
// stdin. The api-host and api-port flags are automatically added if they are not provided in `args`. They
// are set to the values of `s.Host` and `s.Port` respectively.
func (s *BaseSuite) ExecuteTestCobraCommandWithStdin(stdin io.Reader, args ...string) (c *cobra.Command, output string, err error) {
	buf := new(bytes.Buffer)
	root := cli.NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(stdin)

	arguments := []string{}
	if !slices.Contains(args, "--api-host") {
		arguments = append(arguments, "--api-host", s.Host)
	}

	if !slices.Contains(args, "--api-port") {
		arguments = append(arguments, "--api-port", fmt.Sprintf("%d", s.Port))
	}
	arguments = append(arguments, args...)

	root.SetArgs(arguments)

	// Need to check if we're running in debug mode for VSCode
	// Empty them if they exist
	if (len(os.Args) > 2) && (os.Args[1] == "-test.run") {
		os.Args[1] = ""
		os.Args[2] = ""
	}

	s.T().Logf("Command to execute: %v", arguments)

	c, err = root.ExecuteC()
	return c, buf.String(), err
}

// ExecuteTestCobraCommandWithStdinBytes executes a cobra command with the given arguments and with a specific
// stdin bytes. The api-host and api-port flags are automatically added if they are not provided in `args`. They
// are set to the values of `s.Host` and `s.Port` respectively.
func (s *BaseSuite) ExecuteTestCobraCommandWithStdinBytes(stdin []byte, args ...string) (c *cobra.Command, output string, err error) {
	return s.ExecuteTestCobraCommandWithStdin(bytes.NewReader(stdin), args...)
}

package cmdtesting

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"

	"github.com/bacalhau-project/bacalhau/cmd/cli"
	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/test/teststack"
	"github.com/stretchr/testify/suite"
)

type BaseNATSSuite struct {
	suite.Suite
	Node     *node.Node
	ClientV2 clientv2.API
	Host     string
	Port     uint16
}

// before each test
func (s *BaseNATSSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	util.Fatal = util.FakeFatalErrorHandler

	computeConfig, err := node.NewComputeConfigWith(node.ComputeConfigParams{
		JobSelectionPolicy: node.JobSelectionPolicy{
			Locality: semantic.Anywhere,
		},
		LocalPublisher: types.LocalPublisherConfig{
			Address: "127.0.0.1",
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
		devstack.WithNetworkType("nats"),
		teststack.WithNoopExecutor(noop_executor.ExecutorConfig{}),
	)
	s.Node = stack.Nodes[0]
	s.Host = s.Node.APIServer.Address
	s.Port = s.Node.APIServer.Port
	s.ClientV2 = clientv2.New(fmt.Sprintf("http://%s:%d", s.Host, s.Port))

	fmt.Println(s.Host, s.Port)
}

// After each test
func (s *BaseNATSSuite) TearDownTest() {
	util.Fatal = util.FakeFatalErrorHandler
	if s.Node != nil {
		s.Node.CleanupManager.Cleanup(context.Background())
	}
}

// ExecuteTestCobraCommand executes a cobra command with the given arguments. The api-host and api-port
// flags are automatically added if they are not provided in `args`. They are set to the values of
// `s.Host` and `s.Port` respectively.
func (s *BaseNATSSuite) ExecuteTestCobraCommand(args ...string) (c *cobra.Command, output string, err error) {
	return s.ExecuteTestCobraCommandWithStdin(nil, args...)
}

// ExecuteTestCobraCommandWithStdin executes a cobra command with the given arguments and with a specific
// stdin. The api-host and api-port flags are automatically added if they are not provided in `args`. They
// are set to the values of `s.Host` and `s.Port` respectively.
func (s *BaseNATSSuite) ExecuteTestCobraCommandWithStdin(stdin io.Reader, args ...string) (c *cobra.Command, output string, err error) {
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

	util.TestError = nil
	c, err = root.ExecuteC()
	if err == nil {
		err = util.TestError
	}
	return c, buf.String(), err
}

// ExecuteTestCobraCommandWithStdinBytes executes a cobra command with the given arguments and with a specific
// stdin bytes. The api-host and api-port flags are automatically added if they are not provided in `args`. They
// are set to the values of `s.Host` and `s.Port` respectively.
func (s *BaseNATSSuite) ExecuteTestCobraCommandWithStdinBytes(stdin []byte, args ...string) (c *cobra.Command, output string, err error) {
	return s.ExecuteTestCobraCommandWithStdin(bytes.NewReader(stdin), args...)
}

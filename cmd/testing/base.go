package cmdtesting

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/cmd/cli"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/test/teststack"
)

var TestError error

type BaseSuite struct {
	suite.Suite
	Node            *node.Node
	ClientV2        clientv2.API
	Config          types.Bacalhau
	Host            string
	Port            uint16
	AllowListedPath string
}

// before each test
func (s *BaseSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	// disable update checks in testing.
	s.T().Setenv(config.KeyAsEnvVar(types.UpdateConfigIntervalKey), "0")
	// don't send analytics data during testing
	s.T().Setenv(config.KeyAsEnvVar(types.DisableAnalyticsKey), "true")

	var err error
	s.Config, err = config.NewTestConfig()
	s.Require().NoError(err)

	s.AllowListedPath = s.T().TempDir()

	ctx := context.Background()
	stack := teststack.Setup(ctx, s.T(),
		devstack.WithNumberOfHybridNodes(1),
		devstack.WithAllowListedLocalPaths([]string{s.AllowListedPath}),
		teststack.WithNoopExecutor(noop_executor.ExecutorConfig{}, s.Config.Engines),
	)
	s.Node = stack.Nodes[0]
	s.Host = s.Node.APIServer.Address
	s.Port = s.Node.APIServer.Port
	s.ClientV2 = clientv2.New(fmt.Sprintf("http://%s:%d", s.Host, s.Port))
}

// After each test
func (s *BaseSuite) TearDownTest() {
	if s.Node != nil {
		s.Node.CleanupManager.Cleanup(context.Background())
	}
}

// Execute executes a cobra command with the given arguments. The api-host and api-port
// flags are automatically added if they are not provided in `args`. They are set to the values of
// `s.Host` and `s.Port` respectively. The stdout and stderr of the command are returned as well as
// any error that occurred while executing the command.
func (s *BaseSuite) Execute(args ...string) (stdout string, stderr string, err error) {
	stdoutBuf := new(bytes.Buffer)
	stderrBuf := new(bytes.Buffer)
	root := cli.NewRootCmd()
	root.SetOut(stdoutBuf)
	root.SetErr(stderrBuf)

	arguments := []string{}
	if !slices.Contains(args, "--api-host") {
		arguments = append(arguments, "--api-host", s.Host)
	}

	if !slices.Contains(args, "--api-port") {
		arguments = append(arguments, "--api-port", fmt.Sprintf("%d", s.Port))
	}
	arguments = append(arguments, args...)
	root.SetArgs(arguments)

	s.T().Logf("Command to execute: %v", arguments)

	_, err = root.ExecuteC()
	if err != nil {
		return "", "", err
	}
	return stdoutBuf.String(), stderrBuf.String(), nil
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
	// TODO(forrest): we should separate the ouputs from a command into different buffers for stderr and sdtout, otherwise
	// log lines and other outputs (like the update checker) will be included in the returned buffer, and commands
	// that make assertions on the output containing specific values, or being marshaller-able to yaml will fail.
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

	TestError = nil
	c, err = root.ExecuteC()
	if err == nil {
		err = TestError
	}
	return c, buf.String(), err
}

// ExecuteTestCobraCommandWithStdinBytes executes a cobra command with the given arguments and with a specific
// stdin bytes. The api-host and api-port flags are automatically added if they are not provided in `args`. They
// are set to the values of `s.Host` and `s.Port` respectively.
func (s *BaseSuite) ExecuteTestCobraCommandWithStdinBytes(stdin []byte, args ...string) (c *cobra.Command, output string, err error) {
	return s.ExecuteTestCobraCommandWithStdin(bytes.NewReader(stdin), args...)
}

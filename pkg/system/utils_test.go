package system

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SystemUtilsSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSystemUtilsSuite(t *testing.T) {
	suite.Run(t, new(SystemUtilsSuite))
}

// Before all suite
func (s *SystemUtilsSuite) SetupAllSuite() {

}

// Before each test
func (s *SystemUtilsSuite) SetupTest() {
	require.NoError(s.T(), InitConfigForTesting())
}

func (s *SystemUtilsSuite) TearDownTest() {
}

func (s *SystemUtilsSuite) TearDownAllSuite() {

}

func (s *SystemUtilsSuite) TestBasicCommandExecution() {
	cmd := "docker"
	args := []string{"run", "ubuntu", "echo", fmt.Sprintf("%s", uuid.New())}
	tmpDir, err := ioutil.TempDir("", "test-bacalhau-command-execution-")
	defer os.RemoveAll(tmpDir)
	if err != nil {
		require.Fail(s.T(), "Could not create temp dir", err)
	}

	stdoutFile := tmpDir + "/stdout"
	stderrFile := tmpDir + "/stderr"
	RunCommandResultsToDisk(cmd, args, stdoutFile, stderrFile)
}

func (s *SystemUtilsSuite) TestInternalCommandExecution() {
	cmd := "docker"
	args := []string{"run", "ubuntu", "echo", fmt.Sprintf("%s", uuid.New())}
	tmpDir, err := ioutil.TempDir("", "test-bacalhau-command-execution-")
	defer os.RemoveAll(tmpDir)
	if err != nil {
		require.Fail(s.T(), "Could not create temp dir", err)
	}

	stdoutFile := tmpDir + "/stdout"
	stderrFile := tmpDir + "/stderr"
	runCommandResultsToDisk(cmd, args, stdoutFile, stderrFile,
		MaxStdoutFileLengthInGB,
		MaxStderrFileLengthInGB,
		MaxStdoutReturnLengthInBytes,
		MaxStderrReturnLengthInBytes)
}

func (s *SystemUtilsSuite) TestInternalCommandExecutionStdoutTooBig() {
	MaxStdoutReturnLengthInBytes = 10 // Make it artificially small for this run

	stdOutstdErrCases := map[string]struct {
		toStderr bool
	}{
		"stdout": {toStderr: false},
		"stderr": {toStderr: true},
	}

	testCases := map[string]struct {
		inputLength                      int
		outputFileExpectedLength         int
		outputPipeVariableExpectedLength int
		truncated                        bool
	}{
		"zeroLength": {inputLength: 0,
			truncated:                        false,
			outputFileExpectedLength:         0,
			outputPipeVariableExpectedLength: 0},
		"oneLength": {inputLength: 1,
			truncated:                        false,
			outputFileExpectedLength:         1,
			outputPipeVariableExpectedLength: 1},
		"maxLengthMinus1": {inputLength: MaxStdoutReturnLengthInBytes - 1,
			truncated:                        false,
			outputFileExpectedLength:         MaxStdoutReturnLengthInBytes - 1,
			outputPipeVariableExpectedLength: MaxStdoutReturnLengthInBytes - 1},
		"maxLength": {inputLength: MaxStdoutReturnLengthInBytes,
			truncated:                        false,
			outputFileExpectedLength:         MaxStdoutReturnLengthInBytes,
			outputPipeVariableExpectedLength: MaxStdoutReturnLengthInBytes},
		"maxLengthPlus1": {inputLength: MaxStdoutReturnLengthInBytes + 1,
			truncated:                        false,
			outputFileExpectedLength:         MaxStdoutReturnLengthInBytes + 1,
			outputPipeVariableExpectedLength: MaxStdoutReturnLengthInBytes + 1},
		"maxLengthTimes10": {inputLength: MaxStdoutReturnLengthInBytes * 10,
			truncated:                        true,
			outputFileExpectedLength:         MaxStdoutReturnLengthInBytes * 10,
			outputPipeVariableExpectedLength: MaxStdoutReturnLengthInBytes},
	}
	cmd := "docker"
	tmpDir, err := ioutil.TempDir("", "test-bacalhau-command-execution-")
	defer os.RemoveAll(tmpDir)
	if err != nil {
		require.Fail(s.T(), "Could not create temp dir", err)
	}

	for outputPipeTestName, stdOutstdErrCase := range stdOutstdErrCases {
		for sizeTestName, tc := range testCases {
			s.T().Run(sizeTestName, func(t *testing.T) {
				args := []string{"run"} // Reset args

				tmpDirForTC := tmpDir + "/" + sizeTestName
				err := os.Mkdir(tmpDirForTC, 0755)
				if err != nil {
					require.Fail(s.T(), "Could not create temp dir", err)
				}
				defer os.RemoveAll(tmpDirForTC)

				args = append(args, "--name", sizeTestName+uuid.NewString(), "--rm")
				args = append(args, "ubuntu")
				args = append(args, "bash", "-c")

				if stdOutstdErrCase.toStderr {
					args = append(args, RepeatedCharactersBashCommandToStderr(tc.inputLength))
				} else {
					args = append(args, RepeatedCharactersBashCommandToStdout(tc.inputLength))
				}

				log.Debug().Msgf("Running command: %s %s", cmd, strings.Join(args, " "))

				stdoutFile := tmpDirForTC + "/stdout"
				stderrFile := tmpDirForTC + "/stderr"
				runResult, err := runCommandResultsToDisk(cmd, args, stdoutFile, stderrFile,
					MaxStdoutFileLengthInGB,
					MaxStderrFileLengthInGB,
					tc.outputPipeVariableExpectedLength,
					tc.outputPipeVariableExpectedLength)
				require.NoError(t, err) // This is the error from the command execution
				require.NotNil(t, runResult)

				var outputPipeBytes []byte
				if stdOutstdErrCase.toStderr {
					outputPipeBytes, err = os.ReadFile(stderrFile)
				} else {
					outputPipeBytes, err = os.ReadFile(stdoutFile)
				}
				outputPipeContents := string(outputPipeBytes)

				require.NoError(s.T(), err)

				require.Equal(s.T(), tc.outputFileExpectedLength,
					len(outputPipeContents),
					"%s file: %s Not Expected Length",
					outputPipeTestName,
					sizeTestName)

				var outputPipeVariable string
				if stdOutstdErrCase.toStderr {
					outputPipeVariable = runResult.STDERR
				} else {
					outputPipeVariable = runResult.STDOUT
				}

				require.Equal(s.T(), tc.outputPipeVariableExpectedLength,
					len(outputPipeVariable),
					"%s file: %s Not Expected Length",
					outputPipeTestName,
					sizeTestName)

			})
		}
	}

}

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
		MaxStdoutFileLengthInBytes,
		MaxStderrFileLengthInBytes,
		MaxStdoutReturnLengthInBytes,
		MaxStderrReturnLengthInBytes)
}

func (s *SystemUtilsSuite) TestInternalCommandExecutionStdoutTooBigForReturn() {
	GenericMaxLengthInBytes := 10 // Make it artificially small for this run

	maxSizeCases := map[string]struct {
		maxStdoutFileSize   int
		maxStderrFileSize   int
		maxStdoutReturnSize int
		maxStderrReturnSize int
	}{
		"MaxStdoutFileSize": {
			maxStdoutFileSize:   GenericMaxLengthInBytes,
			maxStderrFileSize:   int(MaxStderrFileLengthInBytes),
			maxStdoutReturnSize: int(MaxStdoutReturnLengthInBytes),
			maxStderrReturnSize: int(MaxStderrReturnLengthInBytes),
		},
		"MaxStderrFileSize": {
			maxStdoutFileSize:   int(MaxStdoutFileLengthInBytes),
			maxStderrFileSize:   GenericMaxLengthInBytes,
			maxStdoutReturnSize: int(MaxStdoutReturnLengthInBytes),
			maxStderrReturnSize: int(MaxStderrReturnLengthInBytes),
		},
		"MaxStdoutReturnSize": {
			maxStdoutFileSize:   int(MaxStdoutFileLengthInBytes),
			maxStderrFileSize:   int(MaxStderrFileLengthInBytes),
			maxStdoutReturnSize: GenericMaxLengthInBytes,
			maxStderrReturnSize: int(MaxStderrReturnLengthInBytes),
		},
		"MaxStderrReturnSize": {
			maxStdoutFileSize:   int(MaxStdoutFileLengthInBytes),
			maxStderrFileSize:   int(MaxStderrFileLengthInBytes),
			maxStdoutReturnSize: int(MaxStdoutReturnLengthInBytes),
			maxStderrReturnSize: GenericMaxLengthInBytes,
		},
	}

	stdOutstdErrCases := map[string]struct {
		toStderr bool
	}{
		"stdout": {toStderr: false},
		"stderr": {toStderr: true},
	}

	testCases := map[string]struct {
		inputLength    int
		expectedLength int
	}{
		"zeroLength": {inputLength: 0, expectedLength: 0},
		"oneLength":  {inputLength: 1, expectedLength: 1},
		"maxLengthMinus1": {inputLength: GenericMaxLengthInBytes - 1,
			expectedLength: GenericMaxLengthInBytes - 1},
		"maxLength": {inputLength: GenericMaxLengthInBytes,
			expectedLength: GenericMaxLengthInBytes},
		"maxLengthPlus1": {inputLength: GenericMaxLengthInBytes + 1,
			expectedLength: GenericMaxLengthInBytes},
		"maxLengthTimes10": {inputLength: GenericMaxLengthInBytes * 10,
			expectedLength: GenericMaxLengthInBytes},
	}
	cmd := "docker"
	tmpDir, err := ioutil.TempDir("", "test-bacalhau-command-execution-")
	defer os.RemoveAll(tmpDir)
	if err != nil {
		require.Fail(s.T(), "Could not create temp dir", err)
	}

	for maxSizeCaseName, maxSizeCase := range maxSizeCases {
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

					stdOutFileExpectedLength := Min(maxSizeCase.maxStdoutFileSize, tc.expectedLength)
					stdErrFileExpectedLength := Min(maxSizeCase.maxStderrFileSize, tc.expectedLength)
					stdOutReturnExpectedLength := Min(maxSizeCase.maxStdoutReturnSize, tc.expectedLength)
					stdErrReturnExpectedLength := Min(maxSizeCase.maxStderrReturnSize, tc.expectedLength)

					if stdOutstdErrCase.toStderr {
						args = append(args, RepeatedCharactersBashCommandToStderr(tc.inputLength))
						stdOutFileExpectedLength = 0
						stdOutReturnExpectedLength = 0
					} else {
						args = append(args, RepeatedCharactersBashCommandToStdout(tc.inputLength))
						stdErrFileExpectedLength = 0
						stdErrReturnExpectedLength = 0
					}

					log.Debug().Msgf("Running command: %s %s", cmd, strings.Join(args, " "))

					stdoutFile := tmpDirForTC + "/stdout"
					stderrFile := tmpDirForTC + "/stderr"
					runResult, err := runCommandResultsToDisk(cmd,
						args,
						stdoutFile,
						stderrFile,
						stdOutFileExpectedLength,
						stdErrFileExpectedLength,
						stdOutReturnExpectedLength,
						stdErrReturnExpectedLength)
					require.NoError(t, err) // This is the error from the command execution
					require.NotNil(t, runResult)

					stdoutFileBytes, err := ioutil.ReadFile(stdoutFile)
					require.NoError(s.T(), err)
					stdoutFileContents := string(stdoutFileBytes)

					stderrFileBytes, err := ioutil.ReadFile(stderrFile)
					require.NoError(s.T(), err)
					stderrFileContents := string(stderrFileBytes)

					fileStruct := map[string]struct {
						contents       string
						expectedLength int
					}{
						"stdoutFile": {contents: stdoutFileContents,
							expectedLength: stdOutFileExpectedLength},
						"stderrFile": {contents: stderrFileContents,
							expectedLength: stdErrFileExpectedLength},
						"stdoutReturn": {contents: runResult.STDOUT,
							expectedLength: stdOutReturnExpectedLength},
						"stderrReturn": {contents: runResult.STDERR,
							expectedLength: stdErrReturnExpectedLength},
					}
					for fileStructName, fileStructCase := range fileStruct {
						if fileStructCase.expectedLength != len(fileStructCase.contents) {
							fmt.Printf("failed test case: %s %s %s %s", maxSizeCaseName, outputPipeTestName, sizeTestName, fileStructName)
						}

						require.Equal(s.T(), fileStructCase.expectedLength,
							len(fileStructCase.contents),
							"%s-%s-%s %s contents Not Expected Length",
							maxSizeCaseName,
							outputPipeTestName,
							sizeTestName,
							fileStructName,
						)
					}
				})
			}
		}
	}
}

package system

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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
	cmd := "bash"
	args := []string{"-c", "echo", fmt.Sprintf("%s", uuid.New())}
	tmpDir := s.T().TempDir()

	stdoutFile := tmpDir + "/stdout"
	stderrFile := tmpDir + "/stderr"
	RunCommandResultsToDisk(cmd, args, stdoutFile, stderrFile)
}

func (s *SystemUtilsSuite) TestInternalCommandExecution() {
	cmd := "bash"
	args := []string{"-c", "echo", fmt.Sprintf("%s", uuid.New())}
	tmpDir := s.T().TempDir()

	stdoutFile := tmpDir + "/stdout"
	stderrFile := tmpDir + "/stderr"
	runCommandResultsToDisk(cmd, args, stdoutFile, stderrFile,
		MaxStdoutFileLengthInBytes,
		MaxStderrFileLengthInBytes,
		MaxStdoutReturnLengthInBytes,
		MaxStderrReturnLengthInBytes)
}

// Repeats '=' num times. toStderr true == stderr, false == stdout
func repeatedCharactersBashCommand(num int, toStderr bool) string {
	// If going to stderr, we need to use the special bash command to write to stderr
	toStderrString := ""
	if toStderr {
		toStderrString = "| cat 1>&2"
	}

	return fmt.Sprintf(
		`echo -n %s %s`,
		strings.Repeat("=", num),
		toStderrString,
	)
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
	cmd := "bash"
	tmpDir := s.T().TempDir()

	for maxSizeCaseName, maxSizeCase := range maxSizeCases {
		for outputPipeTestName, stdOutstdErrCase := range stdOutstdErrCases {
			for sizeTestName, tc := range testCases {
				s.T().Run(sizeTestName, func(t *testing.T) {
					args := []string{
						"-c",
						repeatedCharactersBashCommand(tc.inputLength, stdOutstdErrCase.toStderr),
					}

					tmpDirForTC := filepath.Join(tmpDir, sizeTestName)
					err := os.Mkdir(tmpDirForTC, 0755)
					require.NoError(s.T(), err, "Could not create temp dir")
					defer os.RemoveAll(tmpDirForTC)

					stdOutFileExpectedLength := Min(maxSizeCase.maxStdoutFileSize, tc.expectedLength)
					stdErrFileExpectedLength := Min(maxSizeCase.maxStderrFileSize, tc.expectedLength)
					stdOutReturnExpectedLength := Min(maxSizeCase.maxStdoutReturnSize, tc.expectedLength)
					stdErrReturnExpectedLength := Min(maxSizeCase.maxStderrReturnSize, tc.expectedLength)

					if stdOutstdErrCase.toStderr {
						stdOutFileExpectedLength = 0
						stdOutReturnExpectedLength = 0
					} else {
						stdErrFileExpectedLength = 0
						stdErrReturnExpectedLength = 0
					}

					log.Debug().Msgf("Running command: %s %s", cmd, strings.Join(args, " "))

					stdoutFile := filepath.Join(tmpDirForTC, "stdout")
					stderrFile := filepath.Join(tmpDirForTC, "stderr")
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

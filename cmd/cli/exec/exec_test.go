//go:build unit || !integration

package exec_test

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/cmd/cli/exec"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type ExecSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestExecSuite(t *testing.T) {
	suite.Run(t, new(ExecSuite))
}

type testCase struct {
	name                  string
	cmdLine               []string
	expectedUnknownArgs   []string
	expectedErrMsg        string
	jobCommand            string
	jobArguments          []string
	numInlinedAttachments int
	numTotalAttachments   int
}

var testcases []testCase = []testCase{
	{
		// bacalhau exec ruby -e "puts 'hello'"
		name:                "no ruby here",
		cmdLine:             []string{"ruby", "-e", "\"puts 'helllo'\""},
		expectedUnknownArgs: []string{},
		expectedErrMsg:      "the job type 'ruby' is not supported",
	},
	{
		// bacalhau exec python --version=3.10 -- -c "import this"
		name:                  "zen of python",
		cmdLine:               []string{"python", "--version=3.10", "--", "-c", "import this"},
		expectedUnknownArgs:   []string{"--version=3.10", "-c=import this"},
		expectedErrMsg:        "",
		jobCommand:            "python",
		jobArguments:          []string{"-c", `"import this"`},
		numInlinedAttachments: 0,
		numTotalAttachments:   0,
	},
	{
		// bacalhau exec -i src=http://127.0.0.1/test.csv,dst=/inputs/test.csv python app.py
		name:                  "run a python app",
		cmdLine:               []string{"-i", "src=http://127.0.0.1/test.csv,dst=/inputs/test.csv", "python", "app.py", "-x"},
		expectedUnknownArgs:   []string{"-x"},
		expectedErrMsg:        "",
		jobCommand:            "python",
		jobArguments:          []string{"app.py"},
		numInlinedAttachments: 0,
		numTotalAttachments:   1,
	},
	{
		// bacalhau exec -i src=http://127.0.0.1/test.csv,dst=/inputs/test.csv python app.py
		name:                  "run a python app with some inputs",
		cmdLine:               []string{"-i", "src=http://127.0.0.1/test.csv,dst=/inputs/test.csv", "python", "app.py", "/inputs/test.csv"},
		expectedUnknownArgs:   []string{},
		expectedErrMsg:        "",
		jobCommand:            "python",
		jobArguments:          []string{"app.py", "/inputs/test.csv"},
		numInlinedAttachments: 0,
		numTotalAttachments:   1,
	},
	{
		// bacalhau exec -i src=http://127.0.0.1/test.csv,dst=/inputs/test.csv python app.py --code main.go
		name:                  "run a python app with a local file",
		cmdLine:               []string{"-i", "src=http://127.0.0.1/test.csv,dst=/inputs/test.csv", "python", "app.py", "--code=exec_test.go"},
		expectedUnknownArgs:   []string{},
		expectedErrMsg:        "",
		jobCommand:            "python",
		jobArguments:          []string{"app.py"},
		numInlinedAttachments: 1,
		numTotalAttachments:   2,
	},
	{
		// bacalhau exec -i src=http://127.0.0.1/test.csv,dst=/inputs/test.csv duckdb "select * from /inputs/test.csv"
		name:                  "duckdb",
		cmdLine:               []string{"-i", "src=http://127.0.0.1/test.csv,dst=/inputs/test.csv", "duckdb", "select * from /inputs/test.csv"},
		expectedUnknownArgs:   []string{},
		expectedErrMsg:        "",
		jobCommand:            "duckdb",
		jobArguments:          []string{`"select * from /inputs/test.csv"`},
		numInlinedAttachments: 0,
		numTotalAttachments:   1,
	},
}

func (s *ExecSuite) TestJobPreparation() {
	for _, tc := range testcases {
		s.Run(tc.name, func() {
			options := exec.NewExecOptions()
			cmd := exec.NewCmdWithOptions(options)

			testCaseF := s.testFuncForTestCase(tc)

			cmd.PreRunE = nil
			cmd.PostRunE = nil
			cmd.Run = func(cmd *cobra.Command, cmdArgs []string) {
				unknownArgs := exec.ExtractUnknownArgs(cmd.Flags(), tc.cmdLine)
				s.Require().Equal(tc.expectedUnknownArgs, unknownArgs)

				job, err := exec.PrepareJob(cmd, cmdArgs, unknownArgs, options)
				_ = testCaseF(job, err)
			}

			cmd.SetArgs(tc.cmdLine)
			cmd.Execute()
		})
	}

}

func (s *ExecSuite) testFuncForTestCase(tc testCase) func(*models.Job, error) bool {
	return func(job *models.Job, err error) bool {
		if tc.expectedErrMsg == "" {
			s.Require().NoError(err)
		} else {
			s.Require().Error(err)
			s.Require().Contains(err.Error(), tc.expectedErrMsg)
			return false
		}

		task := job.Task()

		s.Require().Equal(tc.jobCommand, task.Engine.Params["Command"], "command is incorrect")
		s.Require().Equal(tc.jobArguments, task.Engine.Params["Arguments"], "arguments are incorrect")

		var inlineCount = 0
		for _, src := range task.InputSources {
			if src.Source.Type == "inline" {
				inlineCount += 1
			}
		}

		s.Require().Equal(tc.numInlinedAttachments, inlineCount, "wrong number of inline attachments")
		s.Require().Equal(tc.numTotalAttachments, len(task.InputSources), "wrong number of input sources")

		return true
	}
}

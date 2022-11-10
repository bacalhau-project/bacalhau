//go:build !(unit && (windows || darwin))

package devstack

import (
	"fmt"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	"github.com/stretchr/testify/suite"
)

type PublishOnErrorSuite struct {
	scenario.ScenarioRunner
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestPublishOnErrorSuite(t *testing.T) {
	suite.Run(t, new(PublishOnErrorSuite))
}

func (s *PublishOnErrorSuite) TestPublishOnError() {
	stdoutText := "I am a miserable failure"

	testcase := scenario.TestCase{
		Spec: model.Spec{
			Engine:    model.EngineDocker,
			Verifier:  model.VerifierNoop,
			Publisher: model.PublisherIpfs,
			Docker: model.JobSpecDocker{
				Image: "ubuntu",
				Entrypoint: []string{
					"bash", "-c",
					fmt.Sprintf("echo %s && exit 1", stdoutText),
				},
			},
		},
		ResultsChecker: scenario.FileEquals(ipfs.DownloadFilenameStdout, stdoutText+"\n"),
		JobCheckers: []job.CheckStatesFunction{
			job.WaitForJobStates(map[model.JobStateType]int{
				model.JobStateCompleted: 1,
			}),
		},
		Outputs: []model.StorageSpec{},
	}

	s.RunScenario(testcase)
}

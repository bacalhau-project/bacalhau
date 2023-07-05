//go:build unit || !integration

package util

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestJobFactorySuite(t *testing.T) {
	suite.Run(t, new(JobFactorySuite))
}

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type JobFactorySuite struct {
	suite.Suite
}

// Before each test
func (suite *JobFactorySuite) SetupTest() {
	logger.ConfigureTestLogging(suite.T())
}

func (suite *JobFactorySuite) TestRun_DockerJobOutputs() {
	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1},
	}

	for range tests {
		testCids := []struct {
			outputVolumes []model.StorageSpec
			correctLength int
		}{
			{outputVolumes: []model.StorageSpec{{Name: "OUTPUT_NAME", Path: "/outputs"}}, correctLength: 1},
			{outputVolumes: []model.StorageSpec{{Name: "APPLE_1", Path: "/apple"}, {Name: "APPLE_2", Path: "/apple"}}, correctLength: 2},
			{outputVolumes: []model.StorageSpec{{Name: "OUTPUT_NAME", Path: "/outputs"}, {Name: "OUTPUT_NAME_1", Path: "/outputs_1"}}, correctLength: 2},     // Two outputs, one default (and dupe), one not
			{outputVolumes: []model.StorageSpec{{Name: "OUTPUT_NAME_1", Path: "/outputs_1"}}, correctLength: 1},                                              // Correct output flag
			{outputVolumes: []model.StorageSpec{{Name: "OUTPUT_NAME_2", Path: "/outputs_2"}, {Name: "OUTPUT_NAME_3", Path: "/outputs_3"}}, correctLength: 2}, // 2 correct output flags
		}

		for _, tcids := range testCids {
			func() {
				spec, err := MakeSpec(
					WithDockerEngine("", "", []string{}, []string{}, []string{}),
					WithResources("1", "1", "0", "0"),
					WithOutputs(tcids.outputVolumes...),
					WithTimeout(300),
					WithDeal(model.TargetAny, 1, 0, 0),
				)
				j := model.Job{
					APIVersion: model.APIVersionLatest().String(),
					Spec:       spec,
				}

				require.NoError(suite.T(), err, "Error in creating spec - %+v", tcids)
				require.Equal(suite.T(), len(j.Spec.Outputs),
					tcids.correctLength,
					"Length of deal outputs (%d) not the same as expected (%d). %+v",
					len(j.Spec.Outputs),
					tcids.correctLength,
					tcids.outputVolumes,
				)
			}()
		}
	}
}

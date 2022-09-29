package devstack

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/suite"
)

type DeterministicVerifierSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestDeterministicVerifierSuite(t *testing.T) {
	suite.Run(t, new(DeterministicVerifierSuite))
}

// Before all suite
func (suite *DeterministicVerifierSuite) SetupAllSuite() {

}

// Before each test
func (suite *DeterministicVerifierSuite) SetupTest() {
	err := system.InitConfigForTesting()
	require.NoError(suite.T(), err)
}

func (suite *DeterministicVerifierSuite) TearDownTest() {
}

func (suite *DeterministicVerifierSuite) TearDownAllSuite() {

}

// test that the combo driver gives preference to the filecoin unsealed driver
// also that this does not affect normal jobs where the CID resides on the IPFS driver
func (suite *DeterministicVerifierSuite) TestDeterministicVerifier() {
	ctx := context.Background()

	apiSubmitJob := func(
		apiClient *publicapi.APIClient,
		args DeterministicVerifierTestArgs,
	) (string, error) {
		j := &model.Job{}
		j.Spec = model.Spec{
			Engine:    model.EngineDocker,
			Verifier:  model.VerifierDeterministic,
			Publisher: model.PublisherNoop,
			Docker: model.JobSpecDocker{
				Image: "ubuntu:latest",
				Entrypoint: []string{
					`echo hello`,
				},
			},
			Inputs: []model.StorageSpec{
				{
					StorageSource: model.StorageSourceIPFS,
					CID:           "123",
				},
			},
			Outputs: []model.StorageSpec{},
			Sharding: model.JobShardingConfig{
				GlobPattern: "/data/*.txt",
				BatchSize:   1,
			},
		}

		j.Deal = model.Deal{
			Concurrency: args.NodeCount,
			Confidence:  args.Confidence,
		}

		submittedJob, err := apiClient.Submit(ctx, j, nil)
		if err != nil {
			return "", err
		}
		return submittedJob.ID, nil
	}

	RunDeterministicVerifierTests(ctx, suite.T(), apiSubmitJob)

}

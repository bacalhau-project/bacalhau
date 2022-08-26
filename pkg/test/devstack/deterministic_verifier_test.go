package devstack

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
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
	system.InitConfigForTesting(suite.T())
}

func (suite *DeterministicVerifierSuite) TearDownTest() {
}

func (suite *DeterministicVerifierSuite) TearDownAllSuite() {

}

// test that the combo driver gives preference to the filecoin unsealed driver
// also that this does not affect normal jobs where the CID resides on the IPFS driver
func (suite *DeterministicVerifierSuite) TestDeterministicVerifier() {

	apiSubmitJob := func(
		apiClient *publicapi.APIClient,
		args DeterministicVerifierTestArgs,
	) (string, error) {
		jobSpec := executor.JobSpec{
			Engine:    executor.EngineDocker,
			Verifier:  verifier.VerifierDeterministic,
			Publisher: publisher.PublisherNoop,
			Docker: executor.JobSpecDocker{
				Image: "ubuntu:latest",
				Entrypoint: []string{
					`echo hello`,
				},
			},
			Inputs: []storage.StorageSpec{
				{
					Engine: storage.StorageSourceIPFS,
					Cid:    "123",
				},
			},
			Outputs: []storage.StorageSpec{},
			Sharding: executor.JobShardingConfig{
				GlobPattern: "/data/*.txt",
				BatchSize:   1,
			},
		}

		jobDeal := executor.JobDeal{
			Concurrency: args.NodeCount,
			Confidence:  args.Confidence,
		}

		submittedJob, err := apiClient.Submit(context.Background(), jobSpec, jobDeal, nil)
		if err != nil {
			return "", err
		}
		return submittedJob.ID, nil
	}

	RunDeterministicVerifierTests(suite.T(), apiSubmitJob)

}

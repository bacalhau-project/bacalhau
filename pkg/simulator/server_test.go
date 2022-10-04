package simulator

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/localdb/inmemory"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SimulatorSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSimulatorSuite(t *testing.T) {
	suite.Run(t, new(SimulatorSuite))
}

// Before all suite
func (suite *SimulatorSuite) SetupAllSuite() {

}

// Before each test
func (suite *SimulatorSuite) SetupTest() {
	err := system.InitConfigForTesting()
	require.NoError(suite.T(), err)
}

func (suite *SimulatorSuite) TearDownTest() {
}

func (suite *SimulatorSuite) TearDownAllSuite() {

}

// Test that the combo driver gives preference to the filecoin unsealed driver
// also that this does not affect normal jobs where the CID resides on the IPFS driver
func (suite *SimulatorSuite) TestSimulatorSanity() {
	nodeCount := 3
	scenario := scenario.CatFileToVolume()

	system.InitConfigForTesting()
	ctx := context.Background()
	cm := system.NewCleanupManager()

	simulatorPort, err := freeport.GetFreePort()
	require.NoError(suite.T(), err)

	localDB, err := inmemory.NewInMemoryDatastore()
	require.NoError(suite.T(), err)

	simulatorServer := NewServer(ctx, "0.0.0.0", simulatorPort, localDB)
	go simulatorServer.ListenAndServe(ctx, cm)

	time.Sleep(time.Second * 1)

	options := devstack.DevStackOptions{
		NumberOfNodes: nodeCount,
		SimulatorURL:  fmt.Sprintf("ws://127.0.0.1:%d/websocket", simulatorPort),
	}

	stack, err := devstack.NewStandardDevStack(ctx, cm, options, computenode.NewDefaultComputeNodeConfig())
	require.NoError(suite.T(), err)

	time.Sleep(time.Second)

	defer cm.Cleanup()

	nodeIDs, err := stack.GetNodeIds()
	require.NoError(suite.T(), err)

	inputStorageList, err := scenario.SetupStorage(ctx, model.StorageSourceIPFS, devstack.ToIPFSClients(stack.Nodes[:nodeCount])...)
	require.NoError(suite.T(), err)

	j := &model.Job{}
	j.Spec = model.Spec{
		Engine:    model.EngineDocker,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherIpfs,
		Docker:    scenario.GetJobSpec(),
		Inputs:    inputStorageList,
		Outputs:   scenario.Outputs,
	}

	j.Deal = model.Deal{
		Concurrency: nodeCount,
	}
	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)
	submittedJob, err := apiClient.Submit(ctx, j, nil)
	require.NoError(suite.T(), err)

	resolver := apiClient.GetJobStateResolver()

	err = resolver.Wait(
		ctx,
		submittedJob.ID,
		len(nodeIDs),
		job.WaitThrowErrors([]model.JobStateType{
			model.JobStateCancelled,
			model.JobStateError,
		}),
		job.WaitForJobStates(map[model.JobStateType]int{
			model.JobStateCompleted: len(nodeIDs),
		}),
	)
	require.NoError(suite.T(), err)
}

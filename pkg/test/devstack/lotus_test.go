//go:build integration

package devstack

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/requesternode"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/publisher/filecoin_lotus/api"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type lotusNodeSuite struct {
	suite.Suite
}

func TestLotusNodeSuite(t *testing.T) {
	suite.Run(t, new(lotusNodeSuite))
}

func (s *lotusNodeSuite) SetupTest() {
	testutils.MustHaveDocker(s.T())

	logger.ConfigureTestLogging(s.T())
	require.NoError(s.T(), system.InitConfigForTesting())
}

func (s *lotusNodeSuite) TestLotusNode() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	testCase := scenario.WasmHelloWorld
	nodeCount := 1

	stack, _ := SetupTest(ctx, s.T(), nodeCount, 0, true, computenode.NewDefaultComputeNodeConfig(), requesternode.NewDefaultRequesterNodeConfig())

	nodeIDs, err := stack.GetNodeIds()
	require.NoError(s.T(), err)

	contextStorageList, err := testCase.Contexts(ctx, model.StorageSourceIPFS, devstack.ToIPFSClients(stack.Nodes[:nodeCount])...)
	require.NoError(s.T(), err)

	j := &model.Job{}
	j.Spec = testCase.Spec
	j.Spec.Verifier = model.VerifierNoop
	j.Spec.Publisher = model.PublisherFilecoin
	j.Spec.Contexts = contextStorageList
	j.Spec.Outputs = testCase.Outputs
	j.Deal = model.Deal{
		Concurrency: 1,
	}

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)
	submittedJob, err := apiClient.Submit(ctx, j, nil)
	require.NoError(s.T(), err)

	resolver := apiClient.GetJobStateResolver()
	resolver.SetWaitTime(10000, time.Millisecond*100)

	err = resolver.Wait(
		ctx,
		submittedJob.ID,
		len(nodeIDs),
		job.WaitThrowErrors([]model.JobStateType{
			model.JobStateError,
		}),
		job.WaitForJobStates(map[model.JobStateType]int{
			model.JobStateCompleted: len(nodeIDs),
		}),
	)
	require.NoError(s.T(), err)

	shards, err := resolver.GetShards(ctx, submittedJob.ID)
	require.NoError(s.T(), err)

	require.NotNil(s.T(), stack.Lotus)
	assert.DirExists(s.T(), stack.Lotus.UploadDir)
	require.DirExists(s.T(), stack.Lotus.PathDir)

	lotus, err := api.NewClientFromConfigDir(ctx, stack.Lotus.PathDir)
	require.NoError(s.T(), err)
	s.T().Cleanup(func() {
		assert.NoError(s.T(), lotus.Close())
	})

	imports, err := lotus.ClientListImports(ctx)
	require.NoError(s.T(), err)

	require.Len(s.T(), imports, 1)
	require.Len(s.T(), shards, 1)

	dir := s.T().TempDir()

	require.NoError(s.T(), ExtractCar(ctx, imports[0].FilePath, dir))

	lotusStdout, err := os.ReadFile(filepath.Join(dir, "stdout"))
	require.NoError(s.T(), err)
	lotusStderr, err := os.ReadFile(filepath.Join(dir, "stderr"))
	require.NoError(s.T(), err)
	lotusExitCodeStr, err := os.ReadFile(filepath.Join(dir, "exitCode"))
	require.NoError(s.T(), err)
	lotusExitCode, err := strconv.Atoi(string(lotusExitCodeStr))
	require.NoError(s.T(), err)

	assert.Equal(s.T(), shards[0].RunOutput.STDOUT, string(lotusStdout))
	assert.Equal(s.T(), shards[0].RunOutput.STDERR, string(lotusStderr))
	assert.Equal(s.T(), shards[0].RunOutput.ExitCode, lotusExitCode)
}

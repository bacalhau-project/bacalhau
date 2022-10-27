//go:build !(unit && (windows || darwin))

package computenode

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/devstack"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ComputeNodeRunJobSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestComputeNodeRunJobSuite(t *testing.T) {
	suite.Run(t, new(ComputeNodeRunJobSuite))
}

// Before all suite
func (s *ComputeNodeRunJobSuite) SetupAllSuite() {

}

// Before each test
func (s *ComputeNodeRunJobSuite) SetupTest() {
	err := system.InitConfigForTesting()
	require.NoError(s.T(), err)
}

func (s *ComputeNodeRunJobSuite) TearDownTest() {

}

func (s *ComputeNodeRunJobSuite) TearDownAllSuite() {

}

// a simple sanity test of the RunJob with docker executor
func (s *ComputeNodeRunJobSuite) TestRunJob() {
	ctx := context.Background()

	tmpOutputDir := s.T().TempDir()

	EXAMPLE_TEXT := "hello"
	stack := testutils.NewDevStack(ctx, s.T(), computenode.NewDefaultComputeNodeConfig())
	computeNode, ipfsStack, cm := stack.Node.ComputeNode, stack.IpfsStack, stack.Node.CleanupManager
	defer cm.Cleanup()

	cid, err := devstack.AddTextToNodes(ctx, []byte(EXAMPLE_TEXT), ipfsStack.IPFSClients[0])
	require.NoError(s.T(), err)

	j := &model.Job{
		ID:   "test",
		Spec: GetJobSpec(cid),
	}
	shard := model.JobShard{
		Job:   j,
		Index: 0,
	}
	runnerOutput, err := computeNode.RunShardExecution(ctx, shard, tmpOutputDir)
	require.NoError(s.T(), err)
	require.Empty(s.T(), runnerOutput.ErrorMsg)

	stdoutPath := fmt.Sprintf("%s/stdout", tmpOutputDir)
	require.FileExists(s.T(), stdoutPath, "The stdout file exists")

	dat, err := os.ReadFile(stdoutPath)
	require.NoError(s.T(), err)
	require.Equal(s.T(), EXAMPLE_TEXT, string(dat), "The stdout file contained the correct result from the job")

}

func (s *ComputeNodeRunJobSuite) TestEmptySpec() {
	ctx := context.Background()
	stack := testutils.NewDevStack(ctx, s.T(), computenode.NewDefaultComputeNodeConfig())
	computeNode, cm := stack.Node.ComputeNode, stack.Node.CleanupManager
	defer cm.Cleanup()

	// it seems when we return an error so quickly we need to sleep a little bit
	// otherwise we don't cleanup
	// TODO: work out why
	time.Sleep(time.Millisecond * 10)
	j := &model.Job{
		ID:   "test",
		Spec: model.Spec{},
	}
	shard := model.JobShard{
		Job:   j,
		Index: 0,
	}
	runnerOutput, err := computeNode.RunShardExecution(ctx, shard, "")
	require.Error(s.T(), err)
	require.Equal(s.T(), runnerOutput.ErrorMsg, err.Error())
}

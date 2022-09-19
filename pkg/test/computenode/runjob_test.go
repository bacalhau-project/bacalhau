package computenode

import (
	"context"
	"fmt"
	"io/ioutil"
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
func (suite *ComputeNodeRunJobSuite) SetupAllSuite() {

}

// Before each test
func (suite *ComputeNodeRunJobSuite) SetupTest() {
	err := system.InitConfigForTesting()
	require.NoError(suite.T(), err)
}

func (suite *ComputeNodeRunJobSuite) TearDownTest() {

}

func (suite *ComputeNodeRunJobSuite) TearDownAllSuite() {

}

// a simple sanity test of the RunJob with docker executor
func (suite *ComputeNodeRunJobSuite) TestRunJob() {
	ctx := context.Background()
	EXAMPLE_TEXT := "hello"
	stack := testutils.NewDockerIpfsStack(ctx, suite.T(), computenode.NewDefaultComputeNodeConfig())
	computeNode, ipfsStack, cm := stack.Node.ComputeNode, stack.IpfsStack, stack.Node.CleanupManager
	defer cm.Cleanup()

	cid, err := devstack.AddTextToNodes(ctx, []byte(EXAMPLE_TEXT), ipfsStack.IPFSClients[0])
	require.NoError(suite.T(), err)

	result, err := ioutil.TempDir("", "bacalhau-TestRunJob")
	require.NoError(suite.T(), err)

	job := model.Job{
		ID:   "test",
		Spec: GetJobSpec(cid),
	}
	shard := model.JobShard{
		Job:   job,
		Index: 0,
	}
	runnerOutput, err := computeNode.RunShardExecution(ctx, shard, result)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), runnerOutput.Error, err)

	stdoutPath := fmt.Sprintf("%s/stdout", result)
	require.FileExists(suite.T(), stdoutPath, "The stdout file exists")

	dat, err := os.ReadFile(stdoutPath)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), EXAMPLE_TEXT, string(dat), "The stdout file contained the correct result from the job")

}

func (suite *ComputeNodeRunJobSuite) TestEmptySpec() {
	ctx := context.Background()
	stack := testutils.NewDockerIpfsStack(ctx, suite.T(), computenode.NewDefaultComputeNodeConfig())
	computeNode, cm := stack.Node.ComputeNode, stack.Node.CleanupManager
	defer cm.Cleanup()

	// it seems when we return an error so quickly we need to sleep a little bit
	// otherwise we don't cleanup
	// TODO: work out why
	time.Sleep(time.Millisecond * 10)
	job := model.Job{
		ID:   "test",
		Spec: model.JobSpec{},
	}
	shard := model.JobShard{
		Job:   job,
		Index: 0,
	}
	runnerOutput, err := computeNode.RunShardExecution(ctx, shard, "")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), runnerOutput.Error, err)
}

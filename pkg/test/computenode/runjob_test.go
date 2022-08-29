package computenode

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
	// noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
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
	system.InitConfigForTesting(suite.T())
}

func (suite *ComputeNodeRunJobSuite) TearDownTest() {
	
}

func (suite *ComputeNodeRunJobSuite) TearDownAllSuite() {

}

// a simple sanity test of the RunJob with docker executor
func (suite *ComputeNodeRunJobSuite) TestRunJob() {
	EXAMPLE_TEXT := "hello"
	computeNode, ipfsStack, cm := SetupTestDockerIpfs(suite.T(), computenode.NewDefaultComputeNodeConfig())
	defer cm.Cleanup()

	cid, err := ipfsStack.AddTextToNodes(1, []byte(EXAMPLE_TEXT))
	require.NoError(suite.T(), err)

	result, err := computeNode.ExecuteJobShard(context.Background(), executor.Job{
		ID:   "test",
		Spec: GetJobSpec(cid),
	}, 0)
	require.NoError(suite.T(), err)

	stdoutPath := fmt.Sprintf("%s/stdout", result)
	require.DirExists(suite.T(), result, "The job result folder exists")
	require.FileExists(suite.T(), stdoutPath, "The stdout file exists")

	dat, err := os.ReadFile(stdoutPath)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), EXAMPLE_TEXT, string(dat), "The stdout file contained the correct result from the job")

}

func (suite *ComputeNodeRunJobSuite) TestEmptySpec() {
	computeNode, _, cm := SetupTestDockerIpfs(suite.T(), computenode.NewDefaultComputeNodeConfig())
	// TODO @enricorotundo #493: replace with SetupTestNoop
	// computeNode, _, _, cm := SetupTestNoop(suite.T(), computenode.NewDefaultComputeNodeConfig(), noop_executor.ExecutorConfig{})
	defer cm.Cleanup()

	// it seems when we return an error so quickly we need to sleep a little bit
	// otherwise we don't cleanup
	// TODO: work out why
	time.Sleep(time.Millisecond * 10)
	_, err := computeNode.ExecuteJobShard(context.Background(), executor.Job{
		ID:   "test",
		Spec: executor.JobSpec{},
	}, 0)
	require.Error(suite.T(), err)
}

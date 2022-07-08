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
	"github.com/stretchr/testify/require"
)

// a simple sanity test of the RunJob with docker executor
func TestRunJob(t *testing.T) {

	EXAMPLE_TEXT := "hello"
	computeNode, ipfsStack, cm := SetupTestDockerIpfs(t, computenode.NewDefaultComputeNodeConfig())
	defer cm.Cleanup()

	cid, err := ipfsStack.AddTextToNodes(1, []byte(EXAMPLE_TEXT))
	require.NoError(t, err)

	result, err := computeNode.RunJob(context.Background(), &executor.Job{
		ID:   "test",
		Spec: GetJobSpec(cid),
	})
	require.NoError(t, err)

	stdoutPath := fmt.Sprintf("%s/stdout", result)
	require.DirExists(t, result, "The job result folder exists")
	require.FileExists(t, stdoutPath, "The stdout file exists")

	dat, err := os.ReadFile(stdoutPath)
	require.NoError(t, err)
	require.Equal(t, EXAMPLE_TEXT, string(dat), "The stdout file contained the correct result from the job")

}

func TestEmptySpec(t *testing.T) {

	computeNode, _, cm := SetupTestDockerIpfs(t, computenode.NewDefaultComputeNodeConfig())
	defer cm.Cleanup()

	// it seems when we return an error so quickly we need to sleep a little bit
	// otherwise we don't cleanup
	// TODO: work out why
	time.Sleep(time.Millisecond * 10)
	_, err := computeNode.RunJob(context.Background(), &executor.Job{
		ID:   "test",
		Spec: nil,
	})
	require.Error(t, err)
}

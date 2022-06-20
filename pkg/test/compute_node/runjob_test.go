package compute_node

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/compute_node"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/stretchr/testify/assert"
)

// a simple sanity test of the RunJob with docker executor
func TestRunJob(t *testing.T) {

	EXAMPLE_TEXT := "hello"
	computeNode, ipfsStack, cm := SetupTest(t, compute_node.JobSelectionPolicy{})
	defer cm.Cleanup()

	cid, err := ipfsStack.AddTextToNodes(1, []byte(EXAMPLE_TEXT))
	assert.NoError(t, err)

	result, err := computeNode.RunJob(context.Background(), &executor.Job{
		Id:   "test",
		Spec: GetJobSpec(cid),
	})
	assert.NoError(t, err)

	stdoutPath := fmt.Sprintf("%s/stdout", result)
	assert.DirExists(t, result, "The job result folder exists")
	assert.FileExists(t, stdoutPath, "The stdout file exists")

	dat, err := os.ReadFile(stdoutPath)
	assert.NoError(t, err)
	assert.Equal(t, EXAMPLE_TEXT, string(dat), "The stdout file contained the correct result from the job")

}

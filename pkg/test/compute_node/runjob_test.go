package devstack

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/compute_node"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/stretchr/testify/assert"
)

// a simple sanity test of the RunJob with docker executor
func TestRunJob(t *testing.T) {

	EXAMPLE_TEXT := "hello"
	ctx := context.Background()

	computeNode, ipfsStack, cm := SetupTest(t, compute_node.JobSelectionPolicy{})
	defer cm.Cleanup()

	cid, err := ipfsStack.AddTextToNodes(1, []byte(EXAMPLE_TEXT))

	jobSpec := &types.JobSpec{
		Engine: string(executor.EXECUTOR_DOCKER),
		Vm: types.JobSpecVm{
			Image: "ubuntu",
			Entrypoint: []string{
				"cat",
				"/test_file.txt",
			},
		},
		Inputs: []types.StorageSpec{
			{
				Engine: "ipfs",
				Cid:    cid,
				Path:   "/test_file.txt",
			},
		},
	}

	result, err := computeNode.RunJob(ctx, &types.Job{
		Id:   "test",
		Spec: jobSpec,
	})
	assert.NoError(t, err)

	stdoutPath := fmt.Sprintf("%s/stdout", result)
	assert.DirExists(t, result, "The job result folder exists")
	assert.FileExists(t, stdoutPath, "The stdout file exists")

	dat, err := os.ReadFile(stdoutPath)
	assert.NoError(t, err)
	assert.Equal(t, EXAMPLE_TEXT, string(dat), "The stdout file contained the correct result from the job")

}

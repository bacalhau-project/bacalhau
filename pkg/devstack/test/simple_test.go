package test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	jobutils "github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestDevStack(t *testing.T) {

	testConcurrency := 3

	stack, cancelFunction := setupTest(
		t,
		3,
		0,
	)

	defer teardownTest(stack, cancelFunction)

	fileCid, err := stack.AddTextToNodes(testConcurrency, []byte(`apple
	orange
	pineapple
	pear
	peach
	cherry
	kiwi is delicious
	strawberry
	lemon
	raspberry
	`))

	assert.NoError(t, err)

	var job *types.Job

	err = system.TryUntilSucceedsN(func() error {
		job, err = jobutils.RunJob(
			"docker",
			[]string{
				fileCid,
			},
			[]string{},
			"ubuntu:latest",
			fmt.Sprintf("grep kiwi /ipfs/%s", fileCid),
			testConcurrency,
			"127.0.0.1",
			stack.Nodes[0].JSONRpcNode.Port,
			true,
		)

		return err
	}, "submit job", 10)

	assert.NoError(t, err)

	// TODO: Do something with the error
	err = system.TryUntilSucceedsN(func() error {
		result, err := jobutils.ListJobs("127.0.0.1", stack.Nodes[0].JSONRpcNode.Port)
		if err != nil {
			return err
		}

		if len(result.Jobs) != 1 {
			return fmt.Errorf("expected 1 job, got %d", len(result.Jobs))
		}

		var jobData *types.Job

		for _, j := range result.Jobs {
			jobData = j
			break
		}

		jobStates := []string{}

		for _, state := range jobData.State {
			jobStates = append(jobStates, state.State)
		}

		if !reflect.DeepEqual(jobStates, []string{"complete", "complete", "complete"}) {
			return fmt.Errorf("expected job to be complete, got %+v", jobStates)
		}

		return nil
	}, "wait for results to be", 100)

	spew.Dump(job)

	// hostId, err := stack.Nodes[0].ComputeNode.Scheduler.HostId()
	// assert.NoError(t, err)

	// resultsDirectory, err := system.GetSystemDirectory(system.GetResultsDirectory(job.Id, hostId))
	// assert.NoError(t, err)

	// stdoutText, err := ioutil.ReadFile(fmt.Sprintf("%s/stdout.log", resultsDirectory))
	// assert.NoError(t, err)

	// assert.True(t, strings.Contains(string(stdoutText), "kiwi is delicious"))
}

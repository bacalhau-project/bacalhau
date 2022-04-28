package test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestDevStack(t *testing.T) {

	executors := map[string]executor.Executor{}
	stack, cancelFunction := setupTest(t, 3, 0, executors)
	defer teardownTest(stack, cancelFunction)

	// create test data
	// ipfs add file on 2 nodes
	// submit job on 1 node
	// wait for job to be done
	// download results and check sanity

	testDir, err := ioutil.TempDir("", "bacalhau-test")
	assert.NoError(t, err)

	testFilePath := fmt.Sprintf("%s/test.txt", testDir)

	dataBytes := []byte(`apple
orange
pineapple
pear
peach
cherry
kiwi is delicious
strawberry
lemon
raspberry
`)

	err = os.WriteFile(testFilePath, dataBytes, 0644)
	assert.NoError(t, err)

	fileCid := ""

	// ipfs add the file to 2 nodes
	// this tests self selection
	for i, node := range stack.Nodes {
		if i >= 2 {
			continue
		}

		fileCid, err = node.IpfsCli.Run([]string{
			"add", "-Q", testFilePath,
		})

		assert.NoError(t, err)
	}

	fmt.Printf("FILE CID: %s\n", fileCid)

	// fileCid = strings.TrimSpace(fileCid)

	// var job *types.Job

	// err = system.TryUntilSucceedsN(func() error {
	// 	job, err = jobutils.RunJob(
	// 		[]string{
	// 			fileCid,
	// 		},
	// 		[]string{},
	// 		"ubuntu:latest",
	// 		fmt.Sprintf("grep kiwi /ipfs/%s", fileCid),
	// 		TEST_CONCURRENCY,
	// 		"127.0.0.1",
	// 		stack.Nodes[0].JSONRpcNode.Port,
	// 		true,
	// 	)

	// 	return err
	// }, "submit job", 100)

	// assert.NoError(t, err)

	// // TODO: Do something with the error
	// err = system.TryUntilSucceedsN(func() error {
	// 	result, err := jobutils.ListJobs("127.0.0.1", stack.Nodes[0].JSONRpcNode.Port)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	if len(result.Jobs) != 1 {
	// 		return fmt.Errorf("expected 1 job, got %d", len(result.Jobs))
	// 	}

	// 	var jobData *types.Job

	// 	for _, j := range result.Jobs {
	// 		jobData = j
	// 		break
	// 	}

	// 	jobStates := []string{}

	// 	for _, state := range jobData.State {
	// 		jobStates = append(jobStates, state.State)
	// 	}

	// 	if !reflect.DeepEqual(jobStates, []string{"complete"}) {
	// 		return fmt.Errorf("expected job to be complete, got %+v", jobStates)
	// 	}

	// 	return nil
	// }, "wait for results to be", 100)

	// hostId, err := stack.Nodes[0].ComputeNode.Scheduler.HostId()
	// assert.NoError(t, err)

	// resultsDirectory, err := system.GetSystemDirectory(system.GetResultsDirectory(job.Id, hostId))
	// assert.NoError(t, err)

	// stdoutText, err := ioutil.ReadFile(fmt.Sprintf("%s/stdout.log", resultsDirectory))
	// assert.NoError(t, err)

	// assert.True(t, strings.Contains(string(stdoutText), "kiwi is delicious"))
}

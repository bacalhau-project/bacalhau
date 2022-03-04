package bacalhau

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/internal"
	"github.com/filecoin-project/bacalhau/internal/ipfs"
	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/filecoin-project/bacalhau/internal/types"
	"github.com/stretchr/testify/assert"
)

func setupTest(t *testing.T) (*internal.DevStack, context.CancelFunc) {
	ctx := context.Background()
	ctxWithCancel, cancelFunction := context.WithCancel(ctx)

	os.Setenv("DEBUG", "true")

	stack, err := internal.NewDevStack(ctxWithCancel, 3)
	assert.NoError(t, err)
	if err != nil {
		log.Fatalf("Unable to create devstack: %s", err)
	}

	// we need a better method for this - i.e. waiting for all the ipfs nodes to be ready
	time.Sleep(time.Second * 2)

	return stack, cancelFunction
}

func teardownTest(stack *internal.DevStack, cancelFunction context.CancelFunc) {
	cancelFunction()
	// need some time to let ipfs processes shut down
	time.Sleep(time.Second * 1)
}

func TestDevStack(t *testing.T) {
	stack, cancelFunction := setupTest(t)
	defer teardownTest(stack, cancelFunction)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			teardownTest(stack, cancelFunction)
			os.Exit(1)
		}
	}()

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

		fileCid, err = ipfs.IpfsCommand(node.IpfsRepo, []string{
			"add", "-Q", testFilePath,
		})

		assert.NoError(t, err)
	}

	fileCid = strings.TrimSpace(fileCid)

	var job *types.Job

	err = system.TryUntilSucceedsN(func() error {
		job, err = SubmitJob([]string{
			fmt.Sprintf("grep kiwi /ipfs/%s", fileCid),
		}, []string{
			fileCid,
		}, "127.0.0.1", stack.Nodes[0].JsonRpcPort)

		return err
	}, "submit job", 100)

	assert.NoError(t, err)

	system.TryUntilSucceedsN(func() error {
		result, err := ListJobs("127.0.0.1", stack.Nodes[0].JsonRpcPort)
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

		if !reflect.DeepEqual(jobStates, []string{"complete", "complete"}) {
			return fmt.Errorf("expected job to be complete, got %+v", jobStates)
		}

		return nil
	}, "wait for results to be", 100)

	hostId, err := stack.Nodes[0].ComputeNode.Scheduler.HostId()
	assert.NoError(t, err)

	resultsDirectory, err := system.GetSystemDirectory(system.GetResultsDirectory(job.Id, hostId))
	assert.NoError(t, err)

	stdoutText, err := ioutil.ReadFile(fmt.Sprintf("%s/stdout.log", resultsDirectory))
	assert.NoError(t, err)

	assert.True(t, strings.Contains(string(stdoutText), "kiwi is delicious"))
}

package bacalhau

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/internal"
	_ "github.com/filecoin-project/bacalhau/internal/logger"

	"github.com/filecoin-project/bacalhau/internal/ipfs"
	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/filecoin-project/bacalhau/internal/traces"
	"github.com/filecoin-project/bacalhau/internal/types"
	"github.com/stretchr/testify/assert"

	"github.com/rs/zerolog/log"
)

// run the job on 2 nodes
const TEST_CONCURRENCY = 1

// both nodes must agree on the result
const TEST_CONFIDENCE = 1

// the results must be within 10% of each other
const TEST_TOLERANCE = 0.1

func setupTest(t *testing.T, nodes int, badActors int) (*internal.DevStack, context.CancelFunc) {
	ctx := context.Background()
	ctxWithCancel, cancelFunction := context.WithCancel(ctx)

	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("BACALHAU_RUNTIME", "docker")

	stack, err := internal.NewDevStack(ctxWithCancel, nodes, badActors)
	assert.NoError(t, err)
	if err != nil {
		log.Fatal().Msg(fmt.Sprintf("Unable to create devstack: %s", err))
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
	stack, cancelFunction := setupTest(t, 1, 0)
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
		if i >= TEST_CONCURRENCY {
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
		job, err = SubmitJob(
			[]string{
				fmt.Sprintf("grep kiwi /ipfs/%s", fileCid),
			},
			[]string{
				fileCid,
			},
			TEST_CONCURRENCY,
			TEST_CONFIDENCE,
			TEST_TOLERANCE,
			"127.0.0.1",
			stack.Nodes[0].JsonRpcPort,
			true,
		)

		return err
	}, "submit job", 100)

	assert.NoError(t, err)

	// TODO: Do something with the error
	err = system.TryUntilSucceedsN(func() error {
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

		if !reflect.DeepEqual(jobStates, []string{"complete"}) {
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

func TestCommands(t *testing.T) {
	tests := map[string]struct {
		file                string
		cmd                 string
		contains            string
		expected_line_count int
	}{
		"grep": {file: "../../testdata/grep_file.txt", cmd: `timeout 2000 grep kiwi /ipfs/%s || echo "ipfs read timed out"`, contains: "kiwi is delicious", expected_line_count: 4},
		// "sed":  {file: "../../testdata/sed_file.txt", cmd: "sed -n '/38.7[2-4]..,-9.1[3-7]../p' /ipfs/%s", contains: "LISBON", expected_line_count: 7},
		// "awk":  {file: "../../testdata/awk_file.txt", cmd: "awk -F',' '{x=38.7077507-$3; y=-9.1365919-$4; if(x^2+y^2<0.3^2) print}' /ipfs/%s", contains: "LISBON", expected_line_count: 7},
	}

	_ = system.RunCommand("sudo", []string{"pkill", "ipfs"})

	stack, cancelFunction := setupTest(t, 3, 0)
	defer teardownTest(stack, cancelFunction)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			teardownTest(stack, cancelFunction)
			os.Exit(1)
		}
	}()

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			log.Warn().Msgf(`
========================================
Starting new job:
  name: %s
   cmd: %s
  file: %s
========================================
`, name, tc.cmd, tc.file)

			// t.Parallel()

			cid, err := add_file_to_nodes(t, stack, tc.file)

			assert.NoError(t, err)

			job, hostId, err := execute_command(t, stack, tc.cmd, cid, TEST_CONCURRENCY, TEST_CONFIDENCE, TEST_TOLERANCE)
			assert.NoError(t, err)

			resultsDirectory, err := system.GetSystemDirectory(system.GetResultsDirectory(job.Id, hostId))
			assert.NoError(t, err)

			stdoutText, err := ioutil.ReadFile(fmt.Sprintf("%s/stdout.log", resultsDirectory))
			assert.NoError(t, err)

			assert.True(t, strings.Contains(string(stdoutText), tc.contains))
			actual_line_count := len(strings.Split(string(stdoutText), "\n"))
			assert.Equal(t, actual_line_count, tc.expected_line_count, fmt.Sprintf("Count mismatch:\nExpected: %d\nActual: %d", tc.expected_line_count, actual_line_count))

		})
	}
}

func add_file_to_nodes(t *testing.T, stack *internal.DevStack, filename string) (string, error) {

	fileCid := ""
	var err error

	// ipfs add the file to 2 nodes
	// this tests self selection
	for i, node := range stack.Nodes {
		if i >= TEST_CONCURRENCY {
			continue
		}

		fileCid, err = ipfs.IpfsCommand(node.IpfsRepo, []string{
			"add", "-Q", filename,
		})
		if err != nil {
			log.Debug().Msgf(`Error running ipfs add -Q: %s`, err)
			return "", err
		}
	}

	fileCid = strings.TrimSpace(fileCid)

	return fileCid, nil
}

func execute_command(
	t *testing.T,
	stack *internal.DevStack,
	cmd string,
	fileCid string,
	concurrency int,
	confidence int,
	tolerance float64,
) (*types.Job, string, error) {
	var job *types.Job
	var err error

	err = system.TryUntilSucceedsN(func() error {

		log.Debug().Msg(fmt.Sprintf(`About to submit job:
cmd: %s`, fmt.Sprintf(cmd, fileCid)))

		job, err = SubmitJob(
			[]string{
				fmt.Sprintf(cmd, fileCid),
			},
			[]string{
				fileCid,
			},
			concurrency,
			confidence,
			tolerance,
			"127.0.0.1",
			stack.Nodes[0].JsonRpcPort,
			true,
		)
		return err
	}, "submit job", 100)

	assert.NoError(t, err)

	// TODO: Do something with the error
	err = system.TryUntilSucceedsN(func() error {
		result, err := ListJobs("127.0.0.1", stack.Nodes[0].JsonRpcPort)
		if err != nil {
			return err
		}

		var jobData *types.Job

		// TODO: Super fragile if executed in parallel.
		for _, j := range result.Jobs {
			jobData = j
		}

		actualJobStates := []string{}
		requiredJobStates := []string{}

		for i := 0; i < concurrency; i++ {
			requiredJobStates = append(requiredJobStates, "complete")
		}

		for _, state := range jobData.State {
			actualJobStates = append(actualJobStates, state.State)
		}

		log.Debug().Msgf("Compare job states:\n%+v\nVS.\n%+v\n", actualJobStates, requiredJobStates)

		if !reflect.DeepEqual(actualJobStates, requiredJobStates) {
			return fmt.Errorf("Expected job states to be %+v, got %+v", requiredJobStates, actualJobStates)
		}

		return nil
	}, "wait for results to be", 100)

	hostId, err := stack.Nodes[0].ComputeNode.Scheduler.HostId()
	assert.NoError(t, err)

	return job, hostId, nil
}

func TestCatchBadActors(t *testing.T) {

	t.Skip()

	tests := map[string]struct {
		nodes       int
		concurrency int
		confidence  int
		tolerance   float64
		badActors   int
		expectation bool
	}{
		"two_agree": {nodes: 3, concurrency: 3, confidence: 2, tolerance: 0.1, badActors: 0, expectation: true},
		// "one_bad_actor": {nodes: 3, concurrency: 2, confidence: 2, tolerance: 0.1, badActors: 1, expectation: false},
	}

	// TODO: #57  This is stupid (for now) but need to add the %s at the end because we don't have an elegant way to run without a cid (yet). Will fix later.
	commands := []string{
		`python3 -c "import time; x = '0'*1024*1024*100; time.sleep(10); %s"`,
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// t.Parallel()

			stack, cancelFunction := setupTest(t, tc.nodes, tc.badActors)
			defer teardownTest(stack, cancelFunction)

			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)
			go func() {
				for range c {
					teardownTest(stack, cancelFunction)
					os.Exit(1)
				}
			}()

			job, _, err := execute_command(t, stack, commands[0], "", tc.concurrency, tc.confidence, tc.tolerance)
			assert.NoError(t, err, "Error executing command: %+v", err)

			resultsList, err := system.ProcessJobIntoResults(job)
			assert.NoError(t, err, "Error processing job into results: %+v", err)

			correctGroup, incorrectGroup, err := traces.ProcessResults(job, resultsList)

			assert.Equal(t, (len(correctGroup)-len(incorrectGroup)) == tc.nodes, fmt.Sprintf("Expected %d good actors, got %d", tc.nodes, len(correctGroup)))
			assert.Equal(t, (len(incorrectGroup)) == tc.badActors, fmt.Sprintf("Expected %d bad actors, got %d", tc.badActors, len(incorrectGroup)))
			assert.NoError(t, err, "Expected to run with no error. Actual: %+v", err)

		})
	}
}

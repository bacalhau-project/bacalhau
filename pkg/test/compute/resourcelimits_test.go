//go:build integration

package compute

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	sync "github.com/bacalhau-project/golang-mutex-tracer"
	"github.com/davecgh/go-spew/spew"
	"github.com/filecoin-project/bacalhau/pkg/compute/capacity"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/system"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ComputeNodeResourceLimitsSuite struct {
	suite.Suite
}

func TestComputeNodeResourceLimitsSuite(t *testing.T) {
	suite.Run(t, new(ComputeNodeResourceLimitsSuite))
}

func (suite *ComputeNodeResourceLimitsSuite) SetupTest() {
	logger.ConfigureTestLogging(suite.T())
	err := system.InitConfigForTesting(suite.T())
	require.NoError(suite.T(), err)
}

type SeenJobRecord struct {
	Id          string
	CurrentJobs int
	MaxJobs     int
	Start       int64
	End         int64
}

type TotalResourceTestCaseCheck struct {
	name    string
	handler func(seenJobs []SeenJobRecord) (bool, error)
}

type TotalResourceTestCase struct {
	// the total list of jobs to throw at the cluster all at the same time
	jobs        []model.ResourceUsageConfig
	totalLimits model.ResourceUsageConfig
	wait        TotalResourceTestCaseCheck
	checkers    []TotalResourceTestCaseCheck
}

func (suite *ComputeNodeResourceLimitsSuite) TestTotalResourceLimits() {

	// for this test we use the transport so the compute_node is calling
	// the executor in a go-routine and we can test what jobs
	// look like over time - this test leave each job running for X seconds
	// and consuming Y resources
	// we will have set a total amount of resources on the compute_node
	// and we want to see that the following things are true:
	//
	//  * all jobs ran eventually (because there is no per job limit and no one job is bigger than the total limit)
	//  * at no time - the total job resource usage exceeds the configured total
	//  * we submit all the jobs at the same time so we prove that compute_nodes "back bid"
	//    * i.e. a job that was seen 20 seconds ago we now have space to run so let's bid on it now
	//
	runTest := func(
		testCase TotalResourceTestCase,
	) {
		ctx := context.Background()

		epochSeconds := time.Now().Unix()

		seenJobs := []SeenJobRecord{}
		var seenJobsMutex sync.Mutex
		seenJobsMutex.EnableTracerWithOpts(sync.Opts{
			Threshold: 10 * time.Millisecond,
			Id:        "TestTotalResourceLimits.seenJobsMutex",
		})

		addSeenJob := func(job SeenJobRecord) {
			seenJobsMutex.Lock()
			defer seenJobsMutex.Unlock()
			seenJobs = append(seenJobs, job)
		}

		currentJobCount := 0
		maxJobCount := 0

		// our function that will "execute the job"
		// record time stamps of start and end
		// sleep for a bit to simulate real work happening
		jobHandler := func(_ context.Context, shard model.JobShard, _ string) (*model.RunCommandResult, error) {
			currentJobCount++
			if currentJobCount > maxJobCount {
				maxJobCount = currentJobCount
			}
			seenJob := SeenJobRecord{
				Id:          shard.Job.Metadata.ID,
				Start:       time.Now().Unix() - epochSeconds,
				CurrentJobs: currentJobCount,
				MaxJobs:     maxJobCount,
			}
			time.Sleep(time.Second * 1)
			currentJobCount--
			seenJob.End = time.Now().Unix() - epochSeconds
			addSeenJob(seenJob)
			return &model.RunCommandResult{}, nil
		}

		getVolumeSizeHandler := func(ctx context.Context, volume model.StorageSpec) (uint64, error) {
			return capacity.ConvertBytesString(volume.CID), nil
		}

		stack := testutils.SetupTestWithNoopExecutor(
			ctx,
			suite.T(),
			devstack.DevStackOptions{NumberOfHybridNodes: 1},
			node.NewComputeConfigWith(node.ComputeConfigParams{
				TotalResourceLimits:           capacity.ParseResourceUsageConfig(testCase.totalLimits),
				IgnorePhysicalResourceLimits:  true,                // in case circleci is running on a small machine
				ExecutorBufferBackoffDuration: 1 * time.Nanosecond, // disable backoff to allow moving from queue to running quickly for this test
			}),
			node.NewRequesterConfigWithDefaults(),
			noop_executor.ExecutorConfig{
				ExternalHooks: noop_executor.ExecutorConfigExternalHooks{
					JobHandler:    jobHandler,
					GetVolumeSize: getVolumeSizeHandler,
				},
			},
		)

		for _, jobResources := range testCase.jobs {
			// what the job is doesn't matter - it will only end up
			j := testutils.MakeNoopJob()
			j.Spec.Resources = jobResources
			j.Spec.Inputs = []model.StorageSpec{
				{
					StorageSource: model.StorageSourceIPFS,
					CID:           jobResources.Disk,
					Name:          "testvolumesize",
				},
			}

			_, err := stack.Nodes[0].RequesterNode.Endpoint.SubmitJob(ctx, model.JobCreatePayload{
				ClientID:   "123",
				APIVersion: j.APIVersion,
				Spec:       &j.Spec,
			})
			require.NoError(suite.T(), err)

			// sleep a bit here to simulate jobs being sumbmitted over time
			time.Sleep((10 + time.Duration(rand.Intn(10))) * time.Millisecond)
		}

		// wait for all the jobs to have completed
		// we can check the seenJobs because that is easier
		waiter := &system.FunctionWaiter{
			Name:        "wait for jobs",
			MaxAttempts: 30,
			Delay:       time.Second * 1,
			Handler: func() (bool, error) {
				return testCase.wait.handler(seenJobs)
			},
		}

		err := waiter.Wait(ctx)
		require.NoError(suite.T(), err, fmt.Sprintf("there was an error in the wait function: %s", testCase.wait.name))

		if err != nil {
			fmt.Printf("error waiting for jobs to have been seen\n")
			spew.Dump(seenJobs)
		}

		checkOk := true
		failingCheckMessage := ""

		for _, checker := range testCase.checkers {
			innerCheck, err := checker.handler(seenJobs)
			errorMessage := ""
			if err != nil {
				errorMessage = fmt.Sprintf("there was an error in the check function: %s %s", checker.name, err.Error())
			}
			require.NoError(suite.T(), err, errorMessage)
			if !innerCheck {
				checkOk = false
				failingCheckMessage = fmt.Sprintf("there was an fail in the check function: %s", checker.name)
			}
		}

		require.True(suite.T(), checkOk, failingCheckMessage)

		if !checkOk {
			fmt.Printf("error checking results on seen jobs\n")
			spew.Dump(seenJobs)
		}
	}

	waitUntilSeenAllJobs := func(expected int) TotalResourceTestCaseCheck {
		return TotalResourceTestCaseCheck{
			name: fmt.Sprintf("waitUntilSeenAllJobs: %d", expected),
			handler: func(seenJobs []SeenJobRecord) (bool, error) {
				return len(seenJobs) >= expected, nil
			},
		}
	}

	checkMaxJobs := func(max int) TotalResourceTestCaseCheck {
		return TotalResourceTestCaseCheck{
			name: fmt.Sprintf("checkMaxJobs: %d", max),
			handler: func(seenJobs []SeenJobRecord) (bool, error) {
				seenMax := 0
				for _, seenJob := range seenJobs {
					if seenJob.MaxJobs > seenMax {
						seenMax = seenJob.MaxJobs
					}
				}
				return seenMax <= max, nil
			},
		}
	}

	// 2 jobs at a time
	// we should end up with 2 groups of 2 in terms of timing
	// and the highest number of jobs at one time should be 2
	suite.Run("2 jobs at a time", func() {
		runTest(
			TotalResourceTestCase{
				jobs: getResourcesArray([][]string{
					{"1", "500Mb", ""},
					{"1", "500Mb", ""},
					{"1", "500Mb", ""},
					{"1", "500Mb", ""},
				}),
				totalLimits: getResources("2", "1Gb", "1Gb"),
				wait:        waitUntilSeenAllJobs(4),
				checkers: []TotalResourceTestCaseCheck{
					// there should only have ever been 2 jobs at one time
					checkMaxJobs(2),
				},
			},
		)
	})

	// test disk space
	// we have a 1Gb disk
	// and 2 jobs each with 900Mb disk space requirements
	// we should only see 1 job at a time
	suite.Run("test disk space", func() {
		runTest(
			TotalResourceTestCase{
				jobs: getResourcesArray([][]string{
					{"100m", "100Mb", "600Mb"},
					{"100m", "100Mb", "600Mb"},
				}),
				totalLimits: getResources("2", "1Gb", "1Gb"),
				wait:        waitUntilSeenAllJobs(2),
				checkers: []TotalResourceTestCaseCheck{
					// there should only have ever been 1 job at one time
					checkMaxJobs(1),
				},
			},
		)
	})

}

// test that with 10 GPU nodes - that 10 jobs end up being allocated 1 per node
// this is a check of the bidding & capacity manager system
func (suite *ComputeNodeResourceLimitsSuite) TestParallelGPU() {
	nodeCount := 2
	jobsPerNode := 2
	seenJobs := 0
	jobIds := []string{}

	ctx := context.Background()

	// the job needs to hang for a period of time so the other job will
	// run on another node
	jobHandler := func(_ context.Context, _ model.JobShard, _ string) (*model.RunCommandResult, error) {
		time.Sleep(time.Millisecond * 1000)
		seenJobs++
		return &model.RunCommandResult{}, nil
	}
	nodeOverrides := make([]node.NodeConfig, nodeCount)
	for i := 0; i < nodeCount; i++ {
		nodeOverrides[i] = node.NodeConfig{
			NodeInfoPublisherInterval: 10 * time.Millisecond, // publish node info quickly for requester node to be aware of compute node infos
		}
	}
	stack := testutils.SetupTestWithNoopExecutor(
		ctx,
		suite.T(),
		devstack.DevStackOptions{NumberOfHybridNodes: nodeCount},
		node.NewComputeConfigWith(node.ComputeConfigParams{
			TotalResourceLimits: model.ResourceUsageData{
				CPU:    1,
				Memory: 1 * 1024 * 1024 * 1024,
				Disk:   1 * 1024 * 1024 * 1024,
				GPU:    1,
			},
			IgnorePhysicalResourceLimits: true, // we need to pretend that we have GPUs on each node

		}),
		node.NewRequesterConfigWithDefaults(),
		noop_executor.ExecutorConfig{
			ExternalHooks: noop_executor.ExecutorConfigExternalHooks{
				JobHandler: jobHandler,
			},
		},
		nodeOverrides...,
	)
	// for the requester node to pick up the nodeInfo messages
	testutils.WaitForNodeDiscovery(suite.T(), stack.Nodes[0], nodeCount)

	jobConfig := &model.Job{
		Spec: model.Spec{
			Engine:    model.EngineNoop,
			Verifier:  model.VerifierNoop,
			Publisher: model.PublisherNoop,
			Resources: model.ResourceUsageConfig{
				GPU: "1",
			},
			Deal: model.Deal{
				Concurrency: 1,
			},
		},
	}

	resolver := job.NewStateResolver(
		func(ctx context.Context, id string) (model.Job, error) {
			return stack.Nodes[0].RequesterNode.JobStore.GetJob(ctx, id)
		},
		func(ctx context.Context, id string) (model.JobState, error) {
			return stack.Nodes[0].RequesterNode.JobStore.GetJobState(ctx, id)
		},
	)

	for i := 0; i < nodeCount; i++ {
		for j := 0; j < jobsPerNode; j++ {
			submittedJob, err := stack.Nodes[0].RequesterNode.Endpoint.SubmitJob(ctx, model.JobCreatePayload{
				ClientID:   "123",
				APIVersion: jobConfig.APIVersion,
				Spec:       &jobConfig.Spec,
			})
			require.NoError(suite.T(), err)
			jobIds = append(jobIds, submittedJob.Metadata.ID)

			// sleep a bit here to simulate jobs being sumbmitted over time
			// this needs to be less than the time the job lasts
			// so we are running jobs in parallel
			time.Sleep((10 + time.Duration(rand.Intn(10))) * time.Millisecond)
		}
	}

	for _, jobId := range jobIds {
		err := resolver.WaitUntilComplete(ctx, jobId)
		require.NoError(suite.T(), err)
	}

	require.Equal(suite.T(), nodeCount*jobsPerNode, seenJobs)

	allocationMap := map[string]int{}

	for _, jobId := range jobIds {
		jobState, err := resolver.GetJobState(ctx, jobId)
		require.NoError(suite.T(), err)
		completedShards := job.GetCompletedShardStates(jobState)
		require.Equal(suite.T(), 1, len(completedShards))
		require.Equal(suite.T(), model.ExecutionStateCompleted, completedShards[0].State)
		allocationMap[completedShards[0].NodeID]++
	}

	// test that each node has 2 job allocated to it
	node1Count, ok := allocationMap[stack.Nodes[0].Host.ID().String()]
	require.True(suite.T(), ok)
	require.Equal(suite.T(), jobsPerNode, node1Count)

	node2Count, ok := allocationMap[stack.Nodes[1].Host.ID().String()]
	require.True(suite.T(), ok)
	require.Equal(suite.T(), jobsPerNode, node2Count)
}

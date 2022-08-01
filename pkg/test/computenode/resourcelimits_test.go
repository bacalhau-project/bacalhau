package computenode

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/filecoin-project/bacalhau/pkg/capacitymanager"
	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/controller"
	devstack "github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	executor_util "github.com/filecoin-project/bacalhau/pkg/executor/util"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/localdb/inmemory"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/inprocess"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	verifier_util "github.com/filecoin-project/bacalhau/pkg/verifier/util"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ComputeNodeResourceLimitsSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestComputeNodeResourceLimitsSuite(t *testing.T) {
	suite.Run(t, new(ComputeNodeResourceLimitsSuite))
}

// Before all suite
func (suite *ComputeNodeResourceLimitsSuite) SetupAllSuite() {

}

// Before each test
func (suite *ComputeNodeResourceLimitsSuite) SetupTest() {
	system.InitConfigForTesting(suite.T())
}

func (suite *ComputeNodeResourceLimitsSuite) TearDownTest() {
}

func (suite *ComputeNodeResourceLimitsSuite) TearDownAllSuite() {

}
func (suite *ComputeNodeResourceLimitsSuite) TestJobResourceLimits() {
	runTest := func(jobResources, jobResourceLimits, defaultJobResourceLimits capacitymanager.ResourceUsageConfig, expectedResult bool) {
		computeNode, _, _, cm := SetupTestNoop(suite.T(), computenode.ComputeNodeConfig{
			CapacityManagerConfig: capacitymanager.Config{
				ResourceLimitJob:            jobResourceLimits,
				ResourceRequirementsDefault: defaultJobResourceLimits,
			},
		}, noop_executor.ExecutorConfig{})
		defer func() {
			// sleep here otherwise the compute node tries to register cleanup handlers too late
			time.Sleep(time.Millisecond * 10)
			cm.Cleanup()
		}()
		job := GetProbeData("")
		job.Spec.Resources = jobResources

		result, _, err := computeNode.SelectJob(context.Background(), job)
		require.NoError(suite.T(), err)

		require.Equal(suite.T(), expectedResult, result, fmt.Sprintf("the expcted result was %v, but got %v -- %+v vs %+v", expectedResult, result, jobResources, jobResourceLimits))
	}

	// the job is half the limit
	runTest(
		getResources("1", "500Mb", ""),
		getResources("2", "1Gb", ""),
		getResources("100m", "100Mb", ""),
		true,
	)

	// the job is on the limit
	runTest(
		getResources("1", "500Mb", ""),
		getResources("1", "500Mb", ""),
		getResources("100m", "100Mb", ""),
		true,
	)

	// the job is over the limit
	runTest(
		getResources("2", "1Gb", ""),
		getResources("1", "500Mb", ""),
		getResources("100m", "100Mb", ""),
		false,
	)

	// test with fractional CPU
	// the job is less than the limit
	runTest(
		getResources("250m", "200Mb", ""),
		getResources("1", "500Mb", ""),
		getResources("100m", "100Mb", ""),
		true,
	)

	// test when the limit is empty
	runTest(
		getResources("250m", "200Mb", ""),
		getResources("", "", ""),
		getResources("100m", "100Mb", ""),
		true,
	)

	// test when both is empty
	runTest(
		getResources("", "", ""),
		getResources("", "", ""),
		getResources("100m", "100Mb", ""),
		true,
	)

	runTest(
		getResources("", "", ""),
		getResources("250m", "200Mb", ""),
		getResources("100m", "100Mb", ""),
		true,
	)

	runTest(
		getResources("300m", "", ""),
		getResources("250m", "200Mb", ""),
		getResources("100m", "100Mb", ""),
		false,
	)

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
	jobs        []capacitymanager.ResourceUsageConfig
	totalLimits capacitymanager.ResourceUsageConfig
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

		epochSeconds := time.Now().Unix()

		seenJobs := []SeenJobRecord{}
		var seenJobsMutex sync.Mutex

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
		jobHandler := func(ctx context.Context, job executor.Job) (string, error) {
			currentJobCount++
			if currentJobCount > maxJobCount {
				maxJobCount = currentJobCount
			}
			seenJob := SeenJobRecord{
				Id:          job.ID,
				Start:       time.Now().Unix() - epochSeconds,
				CurrentJobs: currentJobCount,
				MaxJobs:     maxJobCount,
			}
			time.Sleep(time.Second * 1)
			currentJobCount--
			seenJob.End = time.Now().Unix() - epochSeconds
			addSeenJob(seenJob)
			return "/tmp", nil
		}

		getVolumeSizeHandler := func(ctx context.Context, volume storage.StorageSpec) (uint64, error) {
			return capacitymanager.ConvertMemoryString(volume.Cid), nil
		}

		_, _, ctrl, cm := SetupTestNoop(
			suite.T(),
			computenode.ComputeNodeConfig{
				CapacityManagerConfig: capacitymanager.Config{
					ResourceLimitTotal: testCase.totalLimits,
				},
			},

			noop_executor.ExecutorConfig{

				ExternalHooks: noop_executor.ExecutorConfigExternalHooks{
					JobHandler:    jobHandler,
					GetVolumeSize: getVolumeSizeHandler,
				},
			},
		)
		defer cm.Cleanup()

		for _, jobResources := range testCase.jobs {

			// what the job is doesn't matter - it will only end up
			spec, deal, err := job.ConstructDockerJob(
				executor.EngineNoop,
				verifier.VerifierNoop,
				jobResources.CPU,
				jobResources.Memory,
				"0", // zero GPU for now
				[]string{},
				// pass the disk requirement of the job resources into the volume
				// name so it can be returned from the GetVolumeSize function
				[]string{
					fmt.Sprintf("%s:testvolumesize", jobResources.Disk),
				},
				[]string{},
				[]string{},
				[]string{},
				"",
				1,
				[]string{},
			)

			require.NoError(suite.T(), err)
			_, err = ctrl.SubmitJob(context.Background(), executor.JobCreatePayload{
				ClientID: "123",
				Spec:     spec,
				Deal:     deal,
			})
			require.NoError(suite.T(), err)

			// sleep a bit here to simulate jobs being sumbmitted over time
			time.Sleep((10 + time.Duration(rand.Intn(10))) * time.Millisecond)
		}

		// wait for all the jobs to have completed
		// we can check the seenJobs because that is easier
		waiter := &system.FunctionWaiter{
			Name:        "wait for jobs",
			MaxAttempts: 10,
			Delay:       time.Second * 1,
			Handler: func() (bool, error) {
				return testCase.wait.handler(seenJobs)
			},
		}

		err := waiter.Wait()
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

	// test disk space
	// we have a 1Gb disk
	// and 2 jobs each with 900Mb disk space requirements
	// we should only see 1 job at a time
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

}

func (suite *ComputeNodeResourceLimitsSuite) TestDockerResourceLimitsCPU() {

	CPU_LIMIT := "100m"

	computeNode, _, cm := SetupTestDockerIpfs(suite.T(), computenode.NewDefaultComputeNodeConfig())
	defer cm.Cleanup()

	// this will give us a numerator and denominator that should end up at the
	// same 0.1 value that 100m means
	// https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/8/html/managing_monitoring_and_updating_the_kernel/using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications_managing-monitoring-and-updating-the-kernel#proc_controlling-distribution-of-cpu-time-for-applications-by-adjusting-cpu-bandwidth_using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications
	result := RunJobGetStdout(suite.T(), computeNode, executor.JobSpec{
		Engine:   executor.EngineDocker,
		Verifier: verifier.VerifierNoop,
		Resources: capacitymanager.ResourceUsageConfig{
			CPU:    CPU_LIMIT,
			Memory: "100mb",
		},
		Docker: executor.JobSpecDocker{
			Image: "ubuntu",
			Entrypoint: []string{
				"bash",
				"-c",
				"cat /sys/fs/cgroup/cpu.max",
			},
		},
	})

	values := strings.Fields(result)

	numerator, err := strconv.Atoi(values[0])
	require.NoError(suite.T(), err)

	denominator, err := strconv.Atoi(values[1])
	require.NoError(suite.T(), err)

	var containerCPU float64 = 0

	if denominator > 0 {
		containerCPU = float64(numerator) / float64(denominator)
	}

	require.Equal(suite.T(), capacitymanager.ConvertCPUString(CPU_LIMIT), containerCPU, "the container reported CPU does not equal the configured limit")
}

func (suite *ComputeNodeResourceLimitsSuite) TestDockerResourceLimitsMemory() {

	MEMORY_LIMIT := "100mb"

	computeNode, _, cm := SetupTestDockerIpfs(suite.T(), computenode.NewDefaultComputeNodeConfig())
	defer cm.Cleanup()

	result := RunJobGetStdout(suite.T(), computeNode, executor.JobSpec{
		Engine:   executor.EngineDocker,
		Verifier: verifier.VerifierNoop,
		Resources: capacitymanager.ResourceUsageConfig{
			CPU:    "100m",
			Memory: MEMORY_LIMIT,
		},
		Docker: executor.JobSpecDocker{
			Image: "ubuntu",
			Entrypoint: []string{
				"bash",
				"-c",
				"cat /sys/fs/cgroup/memory.max",
			},
		},
	})

	intVar, err := strconv.Atoi(strings.TrimSpace(result))
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), capacitymanager.ConvertMemoryString(MEMORY_LIMIT), uint64(intVar), "the container reported memory does not equal the configured limit")
}

func (suite *ComputeNodeResourceLimitsSuite) TestDockerResourceLimitsDisk() {

	runTest := func(text, diskSize string, expected bool) {
		computeNode, ipfsStack, cm := SetupTestDockerIpfs(suite.T(), computenode.ComputeNodeConfig{
			CapacityManagerConfig: capacitymanager.Config{
				ResourceLimitTotal: capacitymanager.ResourceUsageConfig{
					// so we have a compute node with 1 byte of disk space
					Disk: diskSize,
				},
			},
		})
		defer cm.Cleanup()

		cid, err := ipfsStack.AddTextToNodes(1, []byte(text))

		result, _, err := computeNode.SelectJob(context.Background(), computenode.JobSelectionPolicyProbeData{
			NodeID: "test",
			JobID:  "test",
			Spec: executor.JobSpec{
				Engine:   executor.EngineDocker,
				Verifier: verifier.VerifierNoop,
				Resources: capacitymanager.ResourceUsageConfig{
					CPU:    "100m",
					Memory: "100mb",
					// we simulate having calculated the disk size here
					Disk: "6b",
				},
				Inputs: []storage.StorageSpec{
					{
						Engine: storage.StorageSourceIPFS,
						Cid:    cid,
						Path:   "/data/file.txt",
					},
				},
				Docker: executor.JobSpecDocker{
					Image: "ubuntu",
					Entrypoint: []string{
						"bash",
						"-c",
						"/data/file.txt",
					},
				},
			},
		})

		require.NoError(suite.T(), err)
		require.Equal(suite.T(), expected, result)
	}

	runTest("hello from 1b test", "1b", false)
	runTest("hello from 1k test", "1k", true)

}

// how many bytes more does ipfs report the file than the actual content?
const IpfsMetadataSize = 8

func (suite *ComputeNodeResourceLimitsSuite) TestGetVolumeSize() {

	runTest := func(text string, expected uint64) {

		cm := system.NewCleanupManager()

		ipfsStack, err := devstack.NewDevStackIPFS(cm, 1)
		require.NoError(suite.T(), err)

		apiAddress := ipfsStack.Nodes[0].IpfsClient.APIAddress()
		transport, err := inprocess.NewInprocessTransport()
		require.NoError(suite.T(), err)

		datastore, err := inmemory.NewInMemoryDatastore()
		require.NoError(suite.T(), err)

		storageProviders, err := executor_util.NewStandardStorageProviders(cm, apiAddress)
		require.NoError(suite.T(), err)

		executors, err := executor_util.NewStandardExecutors(cm, apiAddress, "devstacknode0")
		require.NoError(suite.T(), err)

		verifiers, err := verifier_util.NewIPFSVerifiers(cm, apiAddress)
		require.NoError(suite.T(), err)

		ctrl, err := controller.NewController(cm, datastore, transport, storageProviders)
		require.NoError(suite.T(), err)

		_, err = computenode.NewComputeNode(
			cm,
			ctrl,
			executors,
			verifiers,
			computenode.ComputeNodeConfig{},
		)
		require.NoError(suite.T(), err)

		cid, err := ipfsStack.AddTextToNodes(1, []byte(text))
		require.NoError(suite.T(), err)

		executor := executors[executor.EngineDocker]

		result, err := executor.GetVolumeSize(context.Background(), storage.StorageSpec{
			Engine: storage.StorageSourceIPFS,
			Cid:    cid,
			Path:   "/",
		})

		require.NoError(suite.T(), err)
		require.Equal(suite.T(), expected+IpfsMetadataSize, result)
	}

	runTest("hello from test volume size", 27)
	runTest("hello world", 11)

}

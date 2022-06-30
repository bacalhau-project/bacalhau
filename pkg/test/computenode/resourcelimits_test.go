package computenode

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	"github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/resourceusage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/stretchr/testify/assert"
)

type SeenJobRecord struct {
	Id          string
	CurrentJobs int
	Start       time.Time
	End         time.Time
}

func TestJobResourceLimits(t *testing.T) {
	runTest := func(jobResources, limits resourceusage.ResourceUsageConfig, expectedResult bool) {
		computeNode, _, cm := SetupTestNoop(t, computenode.ComputeNodeConfig{
			JobSelectionPolicy: computenode.JobSelectionPolicy{
				ResourceLimits: limits,
			},
		}, noop_executor.ExecutorConfig{})
		defer cm.Cleanup()
		job := GetProbeData("")
		job.Spec.Resources = jobResources

		result, err := computeNode.SelectJob(context.Background(), job)
		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result, fmt.Sprintf("the expcted result was %v, but got %v -- %+v vs %+v", expectedResult, result, jobResources, limits))
	}

	// the job is half the limit
	runTest(
		getResources("1", "500Mb"),
		getResources("2", "1Gb"),
		true,
	)

	// // the job is on the limit
	runTest(
		getResources("1", "500Mb"),
		getResources("1", "500Mb"),
		true,
	)

	// the job is over the limit
	runTest(
		getResources("2", "1Gb"),
		getResources("1", "500Mb"),
		false,
	)

	// test with fractional CPU
	// the job is less than the limit
	runTest(
		getResources("250m", "200Mb"),
		getResources("1", "500Mb"),
		true,
	)

	// test when the limit is empty
	runTest(
		getResources("250m", "200Mb"),
		getResources("", ""),
		true,
	)

	// test when both is empty
	runTest(
		getResources("", ""),
		getResources("", ""),
		true,
	)

	// test when job is empty
	// but there are limits and so we should not run the job
	runTest(
		getResources("", ""),
		getResources("250m", "200Mb"),
		false,
	)

}

func TestTotalResourceLimits(t *testing.T) {

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
		// the total list of jobs to throw at the cluster all at the same time
		allJobs []resourceusage.ResourceUsageConfig,
		totalLimits resourceusage.ResourceUsageConfig,
	) {

		seenJobs := []SeenJobRecord{}
		currentJobCount := 0

		_, requestorNode, cm := SetupTestNoop(
			t,
			computenode.ComputeNodeConfig{
				ResourceLimits: totalLimits,
			},

			noop_executor.ExecutorConfig{

				// our function that will "execute the job"
				// record time stamps of start and end
				// sleep for a bit to simulate real work happening
				ExternalHooks: &noop_executor.ExecutorConfigExternalHooks{
					JobHandler: func(ctx context.Context, job *executor.Job) (string, error) {
						seenJob := SeenJobRecord{
							Id:          job.ID,
							Start:       time.Now(),
							CurrentJobs: currentJobCount,
						}
						currentJobCount++
						time.Sleep(time.Second * 1)
						currentJobCount--
						seenJob.End = time.Now()
						seenJobs = append(seenJobs, seenJob)
						return "", nil
					},
				},
			},
		)
		defer cm.Cleanup()

		for _, jobResources := range allJobs {

			// what the job is doesn't matter - it will only end up
			spec, deal, err := job.ConstructJob(
				executor.EngineNoop,
				verifier.VerifierNoop,
				jobResources.CPU,
				jobResources.Memory,
				[]string{},
				[]string{},
				[]string{},
				[]string{},
				"",
				1,
				[]string{},
			)

			assert.NoError(t, err)
			_, err = requestorNode.Transport.SubmitJob(context.Background(), spec, deal)
			assert.NoError(t, err)
		}

		// wait for all the jobs to have completed
		// we can check the seenJobs because that is easier
		waiter := &system.FunctionWaiter{
			Name:        "wait for jobs",
			MaxAttempts: 10,
			Delay:       time.Second * 1,
			Handler: func() (bool, error) {
				return len(seenJobs) >= len(allJobs), nil
			},
		}

		err := waiter.Wait()
		assert.NoError(t, err)

		spew.Dump(seenJobs)
	}

	// 2 jobs at a time
	// we should end up with 2 groups of 2 in terms of timing
	// and the highest number of jobs at one time should be 2
	runTest(
		getResourcesArray([][]string{
			{"1", "500Mb"},
			{"1", "500Mb"},
			{"1", "500Mb"},
			{"1", "500Mb"},
		}),
		getResources("2", "1Gb"),
	)

}

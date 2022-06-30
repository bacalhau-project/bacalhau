package computenode

import (
	"context"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/resourceusage"
)

type TotalResourceLimitsTestJobRecord struct {
	Id    string
	Start time.Time
	End   time.Time
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
	) []TotalResourceLimitsTestJobRecord {

		results := []TotalResourceLimitsTestJobRecord{}

		_, _, cm := SetupTestNoop(
			t,
			computenode.ComputeNodeConfig{
				ResourceLimits: totalLimits,
			},

			// this is how we hook into the noop_executor to run our function
			// for the "job" - this means we can keep track of which jobs
			// are currently running
			noop_executor.ExecutorConfig{
				ExternalHooks: &noop_executor.ExecutorConfigExternalHooks{
					JobHandler: func(ctx context.Context, job *executor.Job) (string, error) {
						result := TotalResourceLimitsTestJobRecord{
							Id:    job.ID,
							Start: time.Now(),
						}
						time.Sleep(time.Second * 1)
						result.End = time.Now()
						results = append(results, result)
						return "", nil
					},
				},
			},
		)
		defer cm.Cleanup()

		for _, jobResources := range allJobs {
			job := GetProbeData("")
			job.Spec.Resources = jobResources
			// result, err := computeNode.SelectJob(context.Background(), job)
			// assert.NoError(t, err)
		}

		return results
	}

	// the job is half the limit
	runTest(
		getResourcesArray([][]string{
			{"1", "500Mb"},
			{"1", "500Mb"},
		}),
		getResources("4", "2Gb"),
	)

}

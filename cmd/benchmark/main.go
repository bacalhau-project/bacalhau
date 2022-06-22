package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute_node"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	executor_util "github.com/filecoin-project/bacalhau/pkg/executor/util"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	verifier_util "github.com/filecoin-project/bacalhau/pkg/verifier/util"
	"github.com/rs/zerolog/log"
)

func main() {
	if os.Getenv("HONEYCOMB_KEY") == "" {
		panic("Please set HONEYCOMB_KEY for benchmarking:\n  export HONEYCOMB_KEY=<key>")
	}

	cm := system.NewCleanupManager()
	defer cm.Cleanup()

	numNodes := 3
	numBadNodes := 0
	getExecutors := func(addr string, node int) (
		map[executor.EngineType]executor.Executor, error) {

		return executor_util.NewDockerIPFSExecutors(
			cm, addr, fmt.Sprintf("devstack-node-%d", node))
	}
	getVerifiers := func(addr string, node int) (
		map[verifier.VerifierType]verifier.Verifier, error) {

		return verifier_util.NewIPFSVerifiers(cm, addr)
	}
	stack, err := devstack.NewDevStack(cm, numNodes, numBadNodes,
		getExecutors, getVerifiers,
		compute_node.NewDefaultJobSelectionPolicy())
	if err != nil {
		panic(fmt.Errorf("fatal error while spinning up devstack: %w", err))
	}

	nodeIDs, err := stack.GetNodeIds()
	if err != nil {
		panic(fmt.Errorf("fatal error while getting node IDs: %w", err))
	}

	numJobs := 1000
	var jobs []*executor.Job
	for i := 0; i < numJobs; i++ {
		node := rand.Intn(numNodes)
		apiUri := stack.Nodes[node].ApiServer.GetURI()
		apiClient := publicapi.NewAPIClient(apiUri)

		spec, deal := newJob()
		job, err := apiClient.Submit(context.Background(), spec, deal)
		if err != nil {
			panic(fmt.Errorf("fatal error while submitting job: %w", err))
		}

		log.Info().Msgf("Submitted job '%s'.", job.Id)
		jobs = append(jobs, job)

		// Spread out the jobs over time:
		time.Sleep(100 * time.Millisecond)
	}

	log.Info().Msg("Done submitting jobs, waiting for them to finish...")
	for _, job := range jobs {
		err := stack.WaitForJob(context.Background(), job.Id,
			devstack.WaitForJobAllHaveState(nodeIDs,
				executor.JobStateComplete,
				executor.JobStateBidRejected,
				executor.JobStateError),
		)
		if err != nil {
			panic(fmt.Errorf("fatal error while waiting for job: %w", err))
		}
	}

	log.Info().Msg("Jobs have all completed, cleaning up...")
}

func newJob() (*executor.JobSpec, *executor.JobDeal) {
	return &executor.JobSpec{
			Engine:   executor.EngineDocker,
			Verifier: verifier.VerifierIpfs,
			Vm: executor.JobSpecVm{
				Image:      "ubuntu:latest",
				Entrypoint: []string{"echo", "Hello, world!"},
			},
		}, &executor.JobDeal{
			Concurrency: 1,
		}
}

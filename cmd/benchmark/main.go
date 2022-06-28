package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	executor_util "github.com/filecoin-project/bacalhau/pkg/executor/util"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	verifier_util "github.com/filecoin-project/bacalhau/pkg/verifier/util"
	"github.com/rs/zerolog/log"
)

var NumberOfMillisecondsToSpreadJob = 100 * time.Millisecond

func main() {
	if os.Getenv("HONEYCOMB_KEY") == "" {
		panic("Please set HONEYCOMB_KEY for benchmarking:\n  export HONEYCOMB_KEY=<key>")
	}

	cm := system.NewCleanupManager()
	defer cm.Cleanup()

	numNodes := 3
	numBadNodes := 0
	getExecutors := func(addr string, node int) (map[executor.EngineType]executor.Executor, error) {
		return executor_util.NewDockerIPFSExecutors(cm, addr, fmt.Sprintf("devstack-node-%d", node))
	}
	getVerifiers := func(addr string, node int) (map[verifier.VerifierType]verifier.Verifier, error) {
		return verifier_util.NewIPFSVerifiers(cm, addr)
	}
	stack, err := devstack.NewDevStack(cm, numNodes, numBadNodes,
		getExecutors, getVerifiers,
		computenode.NewDefaultJobSelectionPolicy())
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
		node := randInt(numNodes)
		APIUri := stack.Nodes[node].APIServer.GetURI()
		apiClient := publicapi.NewAPIClient(APIUri)

		spec, deal := newJob()
		job, err := apiClient.Submit(context.Background(), spec, deal)
		if err != nil {
			panic(fmt.Errorf("fatal error while submitting job: %w", err))
		}

		log.Info().Msgf("Submitted job '%s'.", job.ID)
		jobs = append(jobs, job)

		// Spread out the jobs over time:
		time.Sleep(NumberOfMillisecondsToSpreadJob)
	}

	log.Info().Msg("Done submitting jobs, waiting for them to finish...")
	for _, job := range jobs {
		err := stack.WaitForJob(context.Background(), job.ID,
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
			VM: executor.JobSpecVM{
				Image:      "ubuntu:latest",
				Entrypoint: []string{"echo", "Hello, world!"},
			},
		}, &executor.JobDeal{
			Concurrency: 1,
		}
}

func randInt(i int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(i)))
	if err != nil {
		log.Fatal().Msg("could not generate random number")
	}

	return int(n.Int64())
}

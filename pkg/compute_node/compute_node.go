package compute_node

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/scheduler"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/rs/zerolog/log"
)

type ComputeNode struct {
	Ctx       context.Context
	Scheduler scheduler.Scheduler
	Executors map[string]executor.Executor
}

func NewComputeNode(
	ctx context.Context,
	scheduler scheduler.Scheduler,
	executors map[string]executor.Executor,
) (*ComputeNode, error) {

	nodeId, err := scheduler.HostId()

	if err != nil {
		return nil, err
	}

	computeNode := &ComputeNode{
		Ctx:       ctx,
		Scheduler: scheduler,
		Executors: executors,
	}

	scheduler.Subscribe(func(jobEvent *types.JobEvent, job *types.Job) {

		switch jobEvent.EventName {

		// a new job has arrived - decide if we want to bid on it
		case system.JOB_EVENT_CREATED:

			// TODO: #63 We should bail out if we do not fit the execution profile of this machine. E.g., the below:
			// if job.Engine == "docker" && !system.IsDockerRunning() {
			// 	err := fmt.Errorf("Could not execute job - execution engine is 'docker' and the Docker daemon does not appear to be running.")
			// 	log.Warn().Msgf(err.Error())
			// 	return false, err
			// }

			shouldRun, err := computeNode.SelectJob(jobEvent.JobSpec)
			if err != nil {
				log.Error().Msgf("There was an error self selecting: %s %+v", err, jobEvent.JobSpec)
				return
			}
			if shouldRun {
				log.Debug().Msgf("We are bidding on a job: %+v", jobEvent.JobSpec)

				// TODO: Check result of bid job
				err = scheduler.BidJob(jobEvent.JobId)
				if err != nil {
					log.Error().Msgf("Error bidding on job: %+v", err)
				}
				return
			} else {
				log.Debug().Msgf("We ignored a job: %+v", jobEvent.JobSpec)
			}

		// we have been given the goahead to run the job
		case system.JOB_EVENT_BID_ACCEPTED:

			// we only care if the accepted bid is for us
			if jobEvent.NodeId != nodeId {
				return
			}

			log.Debug().Msgf("Bid accepted: Server (id: %s) - Job (id: %s)", nodeId, job.Id)

			outputs, err := computeNode.RunJob(job)

			if err != nil {
				log.Error().Msgf("ERROR running the job: %s %+v", err, job)
				_ = scheduler.ErrorJob(job.Id, fmt.Sprintf("Error running the job: %s", err))
			} else {
				log.Info().Msgf("Completed the job - results: %+v %+v", job, outputs)
				_ = scheduler.SubmitResult(
					job.Id,
					fmt.Sprintf("Got job results: %+v", outputs),
					outputs,
				)
			}
		}
	})

	return computeNode, nil
}

// how this is implemented could be improved
// for example - it should be possible to shell out to a user-defined program or send a HTTP request
// with the detauils of the job (input CIDs, submitter reputation etc)
// that will decide if it's worth doing the job or not
// for now - the rule is "do we have all the input CIDS"
// TODO: allow user probes (http / exec) to be used to decide if we should run the job
func (node *ComputeNode) SelectJob(job *types.JobSpec) (bool, error) {

	// check that we have the executor and it's installed
	executor, err := node.getExecutor(job.Engine)
	if err != nil {
		return false, err
	}

	// Accept jobs where there are no cids specified
	if len(job.Inputs) == 0 {
		return true, nil
	}

	// the inputs we have decided we have
	foundInputs := 0

	for _, input := range job.Inputs {

		// see if the storage engine reports that we have the resource locally
		hasStorage, err := executor.HasStorage(input)
		if err != nil {
			log.Error().Msgf("Error checking for storage resource locality: %s", err.Error())
			return false, err
		}
		if hasStorage {
			foundInputs++
		}
	}

	if foundInputs >= len(job.Inputs) {
		log.Info().Msgf("Found %d of %d inputs - accepting job", foundInputs, len(job.Inputs))
		return true, nil
	} else {
		log.Info().Msgf("Found %d of %d inputs - passing on job", foundInputs, len(job.Inputs))
		return false, nil
	}
}

func (node *ComputeNode) RunJob(job *types.Job) ([]types.StorageSpec, error) {

	// the job states how it would like to collect it's results
	// for example job.Spec.Outputs == [{Engine: "ipfs"}]
	// then we need to produce [{Engine: "ipfs", Cid: "Qm..."}]
	outputs := []types.StorageSpec{}

	// check that we have the executor to run this job
	executor, err := node.getExecutor(job.Spec.Engine)
	if err != nil {
		return outputs, err
	}

	outputs, err = executor.RunJob(job)

	return outputs, nil
}

func (node *ComputeNode) getExecutor(engine string) (executor.Executor, error) {
	if _, ok := node.Executors[engine]; !ok {
		return nil, fmt.Errorf("No matching executor found on this server: %s.", engine)
	}
	executorEngine := node.Executors[engine]
	installed, err := executorEngine.IsInstalled()
	if err != nil {
		return nil, err
	}
	if !installed {
		return nil, fmt.Errorf("Executor is not installed: %s.", engine)
	}
	return executorEngine, nil
}

// func (node *ComputeNode) RunJob(job *types.Job) (string, error) {

// 	err := node.checkExecutor(job.Spec.Engine)
// 	if err != nil {
// 		return "", err
// 	}

// 	vm, err := runtime.NewRuntime(job)

// 	if err != nil {
// 		return "", err
// 	}

// 	if vm.Kind == "docker" && !system.IsDockerRunning() {
// 		err := fmt.Errorf("Could not execute job - execution engine is 'docker' and the Docker daemon does not appear to be running.")
// 		log.Warn().Msgf(err.Error())
// 		return "", err
// 	}

// 	resultsFolder, err := system.EnsureSystemDirectory(system.GetResultsDirectory(job.Id, hostId))

// 	if err != nil {
// 		return "", err
// 	}

// 	log.Debug().Msgf("Ensured results directory created: %s", resultsFolder)

// 	// Having an issue with this directory not existing, so double confirming here
// 	if _, err := os.Stat(resultsFolder); os.IsNotExist(err) {
// 		log.Warn().Msgf("Expected results directory for job id (%s) to exist, it does not: %s", job.Id, resultsFolder)
// 	} else {
// 		log.Info().Msgf("Results directory for job id (%s) exists: %s", job.Id, resultsFolder)
// 	}

// 	// we are in private ipfs network mode if we have got a folder path for our repo
// 	err = vm.EnsureIpfsSidecarRunning(node.IpfsRepo)

// 	if err != nil {
// 		return "", err
// 	}

// 	err = vm.RunJob(resultsFolder)

// 	if err != nil {
// 		return "", err
// 	}

// 	resultCid, err := ipfs.AddFolder(node.IpfsRepo, resultsFolder)

// 	if err != nil {
// 		return "", err
// 	}

// 	return resultCid, nil
// }

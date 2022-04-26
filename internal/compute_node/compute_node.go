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
	Executors map[string]executor.Executor
	Scheduler scheduler.Scheduler
	Ctx       context.Context
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

			log.Debug().Msgf("Found new job to schedule: \n%+v\n", jobEvent.JobSpec)

			// TODO: #63 We should bail out if we do not fit the execution profile of this machine. E.g., the below:
			// if job.Engine == "docker" && !system.IsDockerRunning() {
			// 	err := fmt.Errorf("Could not execute job - execution engine is 'docker' and the Docker daemon does not appear to be running.")
			// 	log.Warn().Msgf(err.Error())
			// 	return false, err
			// }

			shouldRun, err := computeNode.SelectJob(jobEvent.JobSpec)
			if err != nil {
				log.Error().Msgf("There was an error self selecting: %s\n%+v\n", err, jobEvent.JobSpec)
				return
			}
			if shouldRun {
				log.Debug().Msgf("We are bidding on a job because the data is local! \n%+v\n", jobEvent.JobSpec)

				// TODO: Check result of bid job
				err = scheduler.BidJob(jobEvent.JobId)
				if err != nil {
					log.Error().Msgf("Error bidding on job: %+v", err)
				}
				return
			} else {
				log.Debug().Msgf("We ignored a job because we didn't have the data: \n%+v\n", jobEvent.JobSpec)
			}

		// we have been given the goahead to run the job
		case system.JOB_EVENT_BID_ACCEPTED:

			// we only care if the accepted bid is for us
			if jobEvent.NodeId != nodeId {
				return
			}

			log.Debug().Msgf("BID ACCEPTED. Server (id: %s) - Job (id: %s)", nodeId, job.Id)

			outputs, err := computeNode.RunJob(job)

			if err != nil {
				log.Error().Msgf("ERROR running the job: %s\n%+v\n", err, job)

				// TODO: Check result of Error job
				_ = scheduler.ErrorJob(job.Id, fmt.Sprintf("Error running the job: %s", err))
			} else {
				log.Info().Msgf("Completed the job - results: %+v\n%+v\n", job, outputs)

				// TODO: Check result of submit result
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

// make sure that we can use the given executor engine on this node
func (node *ComputeNode) checkExecutor(engine string) error {
	if _, ok := node.Executors[engine]; !ok {
		return fmt.Errorf("No matching executor found on this server: %s.", engine)
	}
	executorEngine := node.Executors[engine]
	installed, err := executorEngine.IsInstalled()
	if err != nil {
		return err
	}
	if !installed {
		return fmt.Errorf("Executor is not installed: %s.", engine)
	}
	return nil
}

// how this is implemented could be improved
// for example - it should be possible to shell out to a user-defined program or send a HTTP request
// with the detauils of the job (input CIDs, submitter reputation etc)
// that will decide if it's worth doing the job or not
// for now - the rule is "do we have all the input CIDS"
func (node *ComputeNode) SelectJob(job *types.JobSpec) (bool, error) {
	err := node.checkExecutor(job.Engine)
	if err != nil {
		log.Debug().Msgf(err.Error())
		return false, nil
	}

	executorEngine := node.Executors[job.Engine]

	// Accept jobs where there are no cids specified
	if len(job.Inputs) == 0 {
		return true, nil
	}

	// the inputs we have decided we have
	foundInputs := 0

	for _, input := range job.Inputs {
		hasStorage, err := executorEngine.HasStorage(input)
		if err != nil {
			return false, err
		}
		if hasStorage {
			foundInputs++
		}
	}

	if foundInputs >= len(job.Inputs) {
		log.Info().Msgf("Found all inputs - accepting job\n")
		return false, nil
	} else {
		log.Info().Msgf("Found %d of %d inputs - passing on job\n", foundInputs, len(job.Inputs))
		return false, nil
	}
}

func (node *ComputeNode) RunJob(job *types.Job) ([]types.JobStorage, error) {

	outputs := []types.JobStorage{}

	err := node.checkExecutor(job.Spec.Engine)
	if err != nil {
		return outputs, err
	}

	return outputs, nil
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

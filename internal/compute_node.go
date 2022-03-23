package internal

import (
	"context"
	"fmt"
	"os"

	"github.com/filecoin-project/bacalhau/internal/ipfs"
	"github.com/filecoin-project/bacalhau/internal/runtime"
	"github.com/filecoin-project/bacalhau/internal/scheduler"
	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/filecoin-project/bacalhau/internal/types"
	"github.com/rs/zerolog/log"
)

const IGNITE_IMAGE string = "docker.io/binocarlos/bacalhau-ignite-image:latest"

type ComputeNode struct {
	IpfsRepo string
	BadActor bool
	NodeId   string

	Ctx context.Context

	Scheduler scheduler.Scheduler
}

func NewComputeNode(
	ctx context.Context,
	scheduler scheduler.Scheduler,
	badActor bool,
) (*ComputeNode, error) {

	nodeId, err := scheduler.HostId()

	if err != nil {
		return nil, err
	}

	computeNode := &ComputeNode{
		IpfsRepo:  "",
		Ctx:       ctx,
		Scheduler: scheduler,
		BadActor:  badActor,
		NodeId:    nodeId,
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

			log.Debug().Msgf("BID ACCEPTED. Server (id: %s) - Job (id: %s)", computeNode.NodeId, job.Id)

			cid, err := computeNode.RunJob(job)

			if err != nil {
				log.Error().Msgf("ERROR running the job: %s\n%+v\n", err, job)

				// TODO: Check result of Error job
				_ = scheduler.ErrorJob(job.Id, fmt.Sprintf("Error running the job: %s", err))
			} else {
				log.Info().Msgf("Completed the job - results cid: %s\n%+v\n", cid, job)

				results := []types.JobStorage{
					{
						Engine: "ipfs",
						Cid:    cid,
					},
				}

				// TODO: Check result of submit result
				_ = scheduler.SubmitResult(
					job.Id,
					fmt.Sprintf("Got job results cid: %s", cid),
					results,
				)
			}
		}
	})

	return computeNode, nil
}

// how this is implemented could be improved
// for example - it should be possible to shell out to a user-defined program
// that will decide if it's worth doing the job or not
func (node *ComputeNode) SelectJob(job *types.JobSpec) (bool, error) {
	log.Debug().Msgf("Selecting for job with matching CID(s): %s\n", job.Inputs)
	// Accept jobs where there are no cids specified or we have any one of the specified cids
	if len(job.Inputs) == 0 {
		return true, nil
	}
	for _, input := range job.Inputs {

		hasCid, err := ipfs.HasCid(node.IpfsRepo, input.Cid)
		if err != nil {
			return false, err
		}
		if hasCid {
			log.Info().Msgf("CID (%s) found on this server. Accepting job.", job.Inputs)
			return true, nil
		}
	}

	log.Info().Msgf("No matching CIDs found on this server. Passing on job")
	return false, nil
}

// return a CID of the job results when finished
// this is obtained by running "ipfs add -r <results folder>"
func (node *ComputeNode) RunJob(job *types.Job) (string, error) {

	log.Debug().Msgf("Running job on node: %s", node.NodeId)

	// replace the job commands with a sleep because we are a bad actor
	if node.BadActor {
		jobCopy := *job
		specCopy := *job.Spec
		specCopy.Commands = []string{"sleep 10"}
		jobCopy.Spec = &specCopy
		job = &jobCopy
	}

	vm, err := runtime.NewRuntime(job)

	if err != nil {
		return "", err
	}

	if vm.Kind == "docker" && !system.IsDockerRunning() {
		err := fmt.Errorf("Could not execute job - execution engine is 'docker' and the Docker daemon does not appear to be running.")
		log.Warn().Msgf(err.Error())
		return "", err
	}

	hostId, err := node.Scheduler.HostId()

	if err != nil {
		return "", err
	}

	resultsFolder, err := system.EnsureSystemDirectory(system.GetResultsDirectory(job.Id, hostId))

	if err != nil {
		return "", err
	}

	log.Debug().Msgf("Ensured results directory created: %s", resultsFolder)

	// Having an issue with this directory not existing, so double confirming here
	if _, err := os.Stat(resultsFolder); os.IsNotExist(err) {
		log.Warn().Msgf("Expected results directory for job id (%s) to exist, it does not: %s", job.Id, resultsFolder)
	} else {
		log.Info().Msgf("Results directory for job id (%s) exists: %s", job.Id, resultsFolder)
	}

	output, err := vm.Start()

	if err != nil {
		return "", fmt.Errorf(`Error starting VM: 
Output: %s
Error: %s`, output, err)
	}

	//nolint
	defer vm.Stop()

	// we are in private ipfs network mode if we have got a folder path for our repo
	err = vm.PrepareJob(node.IpfsRepo)

	if err != nil {
		return "", err
	}

	err = vm.RunJob(resultsFolder)

	if err != nil {
		return "", err
	}

	resultCid, err := ipfs.AddFolder(node.IpfsRepo, resultsFolder)

	if err != nil {
		return "", err
	}

	return resultCid, nil
}

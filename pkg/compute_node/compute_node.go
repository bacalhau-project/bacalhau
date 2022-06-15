package compute_node

import (
	"context"
	"fmt"
	"sync"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
)

type ComputeNode struct {
	Mutex     sync.Mutex
	Transport transport.Transport
	Executors map[string]executor.Executor
	Verifiers map[string]verifier.Verifier
}

func NewComputeNode(
	transport transport.Transport,
	executors map[string]executor.Executor,
	verifiers map[string]verifier.Verifier,
) (*ComputeNode, error) {
	ctx := context.Background() // TODO: instrument
	nodeId, err := transport.HostID(ctx)

	if err != nil {
		return nil, err
	}

	computeNode := &ComputeNode{
		Transport: transport,
		Verifiers: verifiers,
		Executors: executors,
	}

	transport.Subscribe(ctx, func(jobEvent *types.JobEvent, job *types.Job) {
		switch jobEvent.EventName {

		// a new job has arrived - decide if we want to bid on it
		case system.JOB_EVENT_CREATED:

			// TODO: #63 We should bail out if we do not fit the execution profile of this machine. E.g., the below:
			// if job.Engine == "docker" && !system.IsDockerRunning() {
			// 	err := fmt.Errorf("Could not execute job - execution engine is 'docker' and the Docker daemon does not appear to be running.")
			// 	log.Warn().Msgf(err.Error())
			// 	return false, err
			// }

			shouldRun, err := computeNode.SelectJob(ctx, jobEvent.JobSpec)
			if err != nil {
				log.Error().Msgf("There was an error self selecting: %s %+v", err, jobEvent.JobSpec)
				return
			}
			if shouldRun {
				logger.LogJobEvent(logger.JobEvent{
					Node: nodeId,
					Type: "compute_node:bid",
					Job:  job.Id,
				})
				log.Debug().Msgf("We are bidding on a job: %+v", jobEvent.JobSpec)

				// TODO: Check result of bid job
				err = transport.BidJob(ctx, jobEvent.JobId)
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

			logger.LogJobEvent(logger.JobEvent{
				Node: nodeId,
				Type: "compute_node:run",
				Job:  job.Id,
				Data: job,
			})

			resultFolder, err := computeNode.RunJob(ctx, job)

			if err != nil {
				log.Error().Msgf("Error running the job: %s %+v", err, job)
				_ = transport.ErrorJob(ctx, job.Id, fmt.Sprintf("Error running the job: %s", err))
				return
			}

			verifier, err := computeNode.getVerifier(ctx, job.Spec.Verifier)
			if err != nil {
				log.Error().Msgf("Error geting the verifier for the job: %s %+v", err, job)
				_ = transport.ErrorJob(ctx, job.Id, fmt.Sprintf("Error geting the verifier for the job: %s", err))
				return
			}

			resultValue, err := verifier.ProcessResultsFolder(
				ctx, job, resultFolder)
			if err != nil {
				log.Error().Msgf("Error verifying results: %s %+v", err, job)
				_ = transport.ErrorJob(ctx, job.Id, fmt.Sprintf("Error verifying results: %s", err))
				return
			}

			logger.LogJobEvent(logger.JobEvent{
				Node: nodeId,
				Type: "compute_node:result",
				Job:  job.Id,
				Data: resultValue,
			})

			err = transport.SubmitResult(
				ctx,
				job.Id,
				fmt.Sprintf("Got job result: %s", resultValue),
				resultValue,
			)
			if err != nil {
				log.Error().Msgf("Error submitting result: %s %+v", err, job)
				_ = transport.ErrorJob(ctx, job.Id, fmt.Sprintf("Error running the job: %s", err))
				return
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
func (node *ComputeNode) SelectJob(ctx context.Context,
	job *types.JobSpec) (bool, error) {

	// check that we have the executor and it's installed
	executor, err := node.getExecutor(ctx, job.Engine)
	if err != nil {
		return false, err
	}

	// check that we have the verifier and it's installed
	_, err = node.getVerifier(ctx, job.Verifier)
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
		hasStorage, err := executor.HasStorage(ctx, input)
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

func (node *ComputeNode) RunJob(ctx context.Context, job *types.Job) (
	string, error) {

	// check that we have the executor to run this job
	executor, err := node.getExecutor(ctx, job.Spec.Engine)
	if err != nil {
		return "", err
	}

	return executor.RunJob(ctx, job)
}

func (node *ComputeNode) getExecutor(ctx context.Context, name string) (
	executor.Executor, error) {

	node.Mutex.Lock()
	defer node.Mutex.Unlock()

	if _, ok := node.Executors[name]; !ok {
		return nil, fmt.Errorf("No matching executor found on this server: %s.", name)
	}

	executorEngine := node.Executors[name]
	installed, err := executorEngine.IsInstalled(ctx)
	if err != nil {
		return nil, err
	}
	if !installed {
		return nil, fmt.Errorf("Executor is not installed: %s.", name)
	}

	return executorEngine, nil
}

func (node *ComputeNode) getVerifier(ctx context.Context, name string) (
	verifier.Verifier, error) {

	node.Mutex.Lock()
	defer node.Mutex.Unlock()

	if _, ok := node.Verifiers[name]; !ok {
		return nil, fmt.Errorf("No matching verifier found on this server: %s.", name)
	}

	verifier := node.Verifiers[name]
	installed, err := verifier.IsInstalled(ctx)
	if err != nil {
		return nil, err
	}
	if !installed {
		return nil, fmt.Errorf("Verifier is not installed: %s.", name)
	}

	return verifier, nil
}

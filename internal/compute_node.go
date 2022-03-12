package internal

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/internal/ipfs"
	"github.com/filecoin-project/bacalhau/internal/logger"
	"github.com/filecoin-project/bacalhau/internal/runtime"
	"github.com/filecoin-project/bacalhau/internal/scheduler"
	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/filecoin-project/bacalhau/internal/types"
)

const IGNITE_IMAGE string = "docker.io/binocarlos/bacalhau-ignite-image:latest"

type ComputeNode struct {
	IpfsRepo                string
	IpfsConnectMultiAddress string
	BadActor                bool

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
		IpfsRepo:                "",
		IpfsConnectMultiAddress: "",
		Ctx:                     ctx,
		Scheduler:               scheduler,
		BadActor:                badActor,
	}

	scheduler.Subscribe(func(jobEvent *types.JobEvent, job *types.Job) {

		switch jobEvent.EventName {

		// a new job has arrived - decide if we want to bid on it
		case system.JOB_EVENT_CREATED:

			logger.Debugf("Found new job to schedule: \n%+v\n", jobEvent.JobSpec)

			shouldRun, err := computeNode.SelectJob(jobEvent.JobSpec)
			if err != nil {
				logger.Errorf("There was an error self selecting: %s\n%+v\n", err, jobEvent.JobSpec)
				return
			}
			if shouldRun {
				logger.Debugf("We are bidding on a job because the data is local! \n%+v\n", jobEvent.JobSpec)
				scheduler.BidJob(jobEvent.JobId)
			} else {
				logger.Debugf("We ignored a job because we didn't have the data: \n%+v\n", jobEvent.JobSpec)
			}

		// we have been given the goahead to run the job
		case system.JOB_EVENT_BID_ACCEPTED:

			// we only care if the accepted bid is for us
			if jobEvent.NodeId != nodeId {
				return
			}

			cid, err := computeNode.RunJob(job)

			if err != nil {
				logger.Errorf("ERROR running the job: %s\n%+v\n", err, job)
				scheduler.ErrorJob(job.Id, fmt.Sprintf("Error running the job: %s", err))
			} else {
				logger.Infof("Completed the job - results cid: %s\n%+v\n", cid, job)

				results := []types.JobStorage{
					{
						Engine: "ipfs",
						Cid:    cid,
					},
				}

				scheduler.SubmitResult(
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
	logger.Debugf("Selecting for job with matching CID(s): %s\n", job.Inputs)
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
			logger.Infof("++++++++++\nCID (%s) found on this server. Accepting job.\n+++++++++++", job.Inputs)
			return true, nil
		}
	}

	logger.Infof("---------No matching CIDs found on this server. Passing on job")
	return false, nil
}

// return a CID of the job results when finished
// this is obtained by running "ipfs add -r <results folder>"
func (node *ComputeNode) RunJob(job *types.Job) (string, error) {

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

	hostId, err := node.Scheduler.HostId()

	if err != nil {
		return "", err
	}

	resultsFolder, err := system.EnsureSystemDirectory(system.GetResultsDirectory(job.Id, hostId))
	if err != nil {
		return "", err
	}

	err = vm.Start()

	if err != nil {
		return "", err
	}

	//nolint
	defer vm.Stop()

	err = vm.PrepareJob(node.IpfsConnectMultiAddress)

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

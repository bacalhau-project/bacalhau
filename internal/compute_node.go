package internal

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/internal/ipfs"
	"github.com/filecoin-project/bacalhau/internal/runtime"
	"github.com/filecoin-project/bacalhau/internal/scheduler"
	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/filecoin-project/bacalhau/internal/types"
)

const IGNITE_IMAGE string = "docker.io/binocarlos/bacalhau-ignite-image:latest"

type ComputeNode struct {
	IpfsRepo                string
	IpfsConnectMultiAddress string

	Ctx context.Context

	Scheduler scheduler.Scheduler
}

func NewComputeNode(
	ctx context.Context,
	scheduler scheduler.Scheduler,
) (*ComputeNode, error) {
	node := &ComputeNode{
		IpfsRepo:                "",
		IpfsConnectMultiAddress: "",
		Ctx:                     ctx,
		Scheduler:               scheduler,
	}

	scheduler.Subscribe(func(eventName string, job *types.JobData) {

		switch eventName {

		// a new job has arrived - decide if we want to bid on it
		case system.JOB_EVENT_CREATED:

			fmt.Printf("NEW JOB SEEN: \n%+v\n", job)

			shouldRun, err := node.SelectJob(job)
			if err != nil {
				fmt.Printf("there was an error self selecting: %s\n%+v\n", err, job)
				return
			}
			if shouldRun {
				fmt.Printf("we are bidding on a job because the data is local! \n%+v\n", job)
				scheduler.BidJob(job.Job.Id)
			} else {
				fmt.Printf("we ignored a job because we didn't have the data: \n%+v\n", job)
			}

		// we have been given the goahead to run the job
		case system.JOB_EVENT_BID_ACCEPTED:

			scheduler.UpdateJobState(job.Job.Id, &types.JobState{
				State:  system.JOB_STATE_RUNNING,
				Status: "",
			})

			cid, err := node.RunJob(job.Job)

			if err != nil {
				fmt.Printf("there was an error running the job: %s\n%+v\n", err, job)
				scheduler.UpdateJobState(job.Job.Id, &types.JobState{
					State:  system.JOB_STATE_ERROR,
					Status: fmt.Sprintf("Error running the job: %s", err),
				})

			} else {
				fmt.Printf("we completed the job - results cid: %s\n%+v\n", cid, job)
				scheduler.UpdateJobState(job.Job.Id, &types.JobState{
					State:     system.JOB_STATE_COMPLETE,
					Status:    fmt.Sprintf("Got job results cid: %s", cid),
					ResultCid: cid,
				})
			}

		}

	})

	return node, nil
}

// how this is implemented could be improved
// for example - it should be possible to shell out to a user-defined program
// that will decide if it's worth doing the job or not
func (node *ComputeNode) SelectJob(job *types.JobData) (bool, error) {
	fmt.Printf("--> FilterJob with %s\n", job.Job.Cids)
	// Accept jobs where there are no cids specified or we have any one of the specified cids
	if len(job.Job.Cids) == 0 {
		return true, nil
	}
	for _, cid := range job.Job.Cids {
		hasCid, err := ipfs.HasCid(node.IpfsRepo, cid)
		if err != nil {
			return false, err
		}
		if hasCid {
			return true, nil
		}
	}

	return false, nil
}

// return a CID of the job results when finished
// this is obtained by running "ipfs add -r <results folder>"
func (node *ComputeNode) RunJob(job *types.JobSpec) (string, error) {

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

package internal

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/internal/scheduler"
	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/filecoin-project/bacalhau/internal/types"
)

type RequesterNode struct {
	Ctx context.Context
	// an array of job ids that we are the "requester" for
	RequestedJobIds []string
	Scheduler       scheduler.Scheduler
}

func NewRequesterNode(
	ctx context.Context,
	scheduler scheduler.Scheduler,
) (*RequesterNode, error) {
	node := &RequesterNode{
		Ctx:       ctx,
		Scheduler: scheduler,
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

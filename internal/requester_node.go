package internal

import (
	"context"

	"github.com/filecoin-project/bacalhau/internal/logger"
	"github.com/filecoin-project/bacalhau/internal/scheduler"
	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/filecoin-project/bacalhau/internal/types"
	"github.com/rs/zerolog/log"

	_ "github.com/filecoin-project/bacalhau/internal/logger"
)

type RequesterNode struct {
	Ctx       context.Context
	Scheduler scheduler.Scheduler
}

func NewRequesterNode(
	ctx context.Context,
	scheduler scheduler.Scheduler,
) (*RequesterNode, error) {

	nodeId, err := scheduler.HostId()
	threadLogger := logger.LoggerWithRuntimeInfo(nodeId)

	if err != nil {
		threadLogger.Error().Err(err)
		return nil, err
	}

	requesterNode := &RequesterNode{
		Ctx:       ctx,
		Scheduler: scheduler,
	}

	scheduler.Subscribe(func(jobEvent *types.JobEvent, job *types.Job) {

		// we only care about jobs that we own
		if job.Owner != nodeId {
			return
		}

		switch jobEvent.EventName {

		// a compute node has bid on a job
		// let's decide if we want to accept it or not
		// we would call out to the reputation system
		// we also pay attention to the job deal concurrency setting
		case system.JOB_EVENT_BID:

			bidAccepted, message, err := requesterNode.ConsiderBid(job, jobEvent.NodeId)

			if err != nil {
				threadLogger.Warn().Msgf("There was an error considering bid: %s", err)
				return
			}

			if bidAccepted {
				// TODO: Check result of accept job bid
				err = scheduler.AcceptJobBid(jobEvent.JobId, jobEvent.NodeId)
				if err != nil {
					threadLogger.Error().Err(err)
				}
			} else {
				// TODO: Check result of reject job bid
				err = scheduler.RejectJobBid(jobEvent.JobId, jobEvent.NodeId, message)
				if err != nil {
					threadLogger.Error().Err(err)
				}
			}

		// a compute node has submitted some results
		// let's consult our confidence and tolerance settings
		// to see if we can "accept" these results
		// or if we need to wait for some more results to arrive
		case system.JOB_EVENT_RESULTS:
			err := requesterNode.ProcessResults(job, jobEvent.NodeId)

			if err != nil {
				// TODO: Check result of Error Job for Node
				err = scheduler.ErrorJobForNode(jobEvent.JobId, jobEvent.NodeId, err.Error())
				threadLogger.Error().Err(err)
			}
		}

	})

	return requesterNode, nil
}

// a compute node has bid on the job
// should we accept the bid or not?
func (node *RequesterNode) ConsiderBid(job *types.Job, nodeId string) (bool, string, error) {

	threadLogger := logger.LoggerWithNodeAndJobInfo(nodeId, job.Id)

	concurrency := job.Deal.Concurrency
	threadLogger.Debug().Msgf("Concurrency for this job: %d", concurrency)

	// we are already over-subscribed
	if len(job.Deal.AssignedNodes) >= concurrency {
		threadLogger.Debug().Msgf("Rejected: Job already on enough nodes (Subscribed: %d vs Concurrency: %d", len(job.Deal.AssignedNodes), concurrency)
		return false, "Job is oversubscribed", nil
	}

	// sanity check to not allow a node to bid on a job twice
	alreadyAssigned := false

	for _, assignedNode := range job.Deal.AssignedNodes {
		if assignedNode == nodeId {
			alreadyAssigned = true
		}
	}

	if alreadyAssigned {
		return false, "This node is already assigned", nil
	}

	// TODO: call out to the reputation system to decide if we want this
	// compute node to join our fleet

	return true, "", nil
}

// a compute node has submitted some results
// let's check if we have >= concurrency results in the set
// if we do - then let's see which results can be grouped as the "same"
// if we have a majority in that case - let's mark those results as "accepted"
// (and reject the rest)
func (node *RequesterNode) ProcessResults(job *types.Job, nodeId string) error {

	// before we do anything - let's fetch the results for the given job
	resultsList, err := system.ProcessJobIntoResults(job)

	if err != nil {
		log.Error().Err(err).Msg("Error processing job into results.")
		return err
	}

	for _, result := range *resultsList {
		log.Debug().Msgf("Currently fetching result for %+v", result)
		err = system.FetchJobResult(result)
		if err != nil {
			log.Error().Err(err).Msgf("Error fetching job results. Job Node: %s", result.Node)
		}
	}

	// ok the results for this job should now be local
	// let's loop over the "AssignedNodes" and see if we have results for all of them
	// if yes - then we run the analysis on the results
	completedNodes := 0

	for _, assignedNode := range job.Deal.AssignedNodes {
		log.Debug().Msgf("Node %s: %s", assignedNode, job.State[assignedNode].State)
		if job.State[assignedNode].State == system.JOB_STATE_COMPLETE {
			completedNodes = completedNodes + 1
		}
	}

	if completedNodes < job.Deal.Concurrency {
		log.Debug().Msgf("Not enough nodes have completed task. Actual: %d  Needed: %d", completedNodes, job.Deal.Concurrency)
		return nil
	}

	// ok all of the nodes that have been assigned have marked the status as complete
	// let's work out who is "correct" and who is "incorrect"
	// TODO: implement the client side checking here to trigger "results-accepted" and "results-rejected" messages

	return nil
}

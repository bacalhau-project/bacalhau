package internal

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/internal/logger"
	"github.com/filecoin-project/bacalhau/internal/scheduler"
	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/filecoin-project/bacalhau/internal/types"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	_ "github.com/filecoin-project/bacalhau/internal/logger"
)

type RequesterNode struct {
	Scheduler scheduler.Scheduler
}

func NewRequesterNode(
	scheduler scheduler.Scheduler,
) (*RequesterNode, error) {

	nodeId, err := scheduler.HostId()
	threadLogger := logger.LoggerWithRuntimeInfo(nodeId)

	if err != nil {
		threadLogger.Error().Err(err)
		return nil, err
	}

	requesterNode := &RequesterNode{
		Scheduler: scheduler,
	}

	scheduler.Subscribe(func(jobEvent *types.JobEvent, job *types.Job) {
		schedulerContext := context.Background()
		schedulerContext = context.WithValue(schedulerContext, "id", job.Id)

		tracer := otel.GetTracerProvider().Tracer("bacalhau.org") // if not already in scope
		_, nodeSpan := tracer.Start(schedulerContext, fmt.Sprintf("Node Listening: Id %s", nodeId))
		nodeSpan.SetAttributes(attribute.String("NodeId", nodeId))

		// TODO: Why is there NodeId (as a local variable) and jobEvent.NodeId (is the latter the requesting node?)
		nodeSpan.AddEvent(fmt.Sprintf("Event received %s", jobEvent.JobId))
		nodeSpan.SetAttributes(attribute.String("JobId", jobEvent.JobId))

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

			_, nodeBidConsiderSpan := tracer.Start(schedulerContext, "Considering bid")
			nodeBidConsiderSpan.SetAttributes(attribute.String("JobId", jobEvent.JobId))
			bidAccepted, message, err := requesterNode.ConsiderBid(job, jobEvent.NodeId)
			nodeBidConsiderSpan.End()

			if err != nil {
				threadLogger.Warn().Msgf("There was an error considering bid: %s", err)
				return
			}

			if bidAccepted {
				_, nodeBidAcceptingSpan := tracer.Start(schedulerContext, "Accepting job")
				nodeBidAcceptingSpan.SetAttributes(attribute.String("JobId", jobEvent.JobId))

				// TODO: Check result of accept job bid
				err = scheduler.AcceptJobBid(jobEvent.JobId, jobEvent.NodeId)
				if err != nil {
					threadLogger.Error().Err(err)
				}
				nodeBidAcceptingSpan.End()

			} else {
				_, nodeBidRejectingSpan := tracer.Start(schedulerContext, "Rejecting job")
				nodeBidRejectingSpan.SetAttributes(attribute.String("JobId", jobEvent.JobId))

				// TODO: Check result of reject job bid
				err = scheduler.RejectJobBid(jobEvent.JobId, jobEvent.NodeId, message)
				if err != nil {
					threadLogger.Error().Err(err)
				}

				nodeBidRejectingSpan.End()
			}

		// a compute node has submitted some results
		// let's consult our confidence and tolerance settings
		// to see if we can "accept" these results
		// or if we need to wait for some more results to arrive
		case system.JOB_EVENT_RESULTS:
			_, nodeProcessResultsSpan := tracer.Start(schedulerContext, "Processing results job")
			nodeProcessResultsSpan.SetAttributes(attribute.String("JobId", jobEvent.JobId))
			err := requesterNode.ProcessResults(job, jobEvent.NodeId)
			nodeProcessResultsSpan.End()

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

	ctx := context.Background()

	threadLogger := logger.LoggerWithNodeAndJobInfo(nodeId, job.Id)

	tracer := otel.GetTracerProvider().Tracer("bacalhau.org")
	_, considerBidSpan := tracer.Start(ctx, "Considering bid")

	concurrency := job.Deal.Concurrency
	threadLogger.Debug().Msgf("Concurrency for this job: %d", concurrency)

	// we are already over-subscribed
	if len(job.Deal.AssignedNodes) >= concurrency {
		considerBidSpan.AddEvent("Job rejected: Oversubscribed")
		threadLogger.Debug().Msgf("Rejected: Job already on enough nodes (Subscribed: %d vs Concurrency: %d", len(job.Deal.AssignedNodes), concurrency)
		considerBidSpan.End()
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
		considerBidSpan.AddEvent("Job rejected: Node already essigned")
		considerBidSpan.End()
		return false, "This node is already assigned", nil
	}

	// TODO: call out to the reputation system to decide if we want this
	// compute node to join our fleet
	considerBidSpan.AddEvent("Job accepted")
	considerBidSpan.End()
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

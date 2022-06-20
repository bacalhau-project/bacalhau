package requestor_node

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type RequesterNode struct {
	NodeID    string
	Transport transport.Transport
}

func NewRequesterNode(
	transport transport.Transport,
) (*RequesterNode, error) {
	ctx := context.Background() // TODO: instrument with trace

	nodeId, err := transport.HostID(ctx)
	threadLogger := logger.LoggerWithRuntimeInfo(nodeId)

	if err != nil {
		threadLogger.Error().Err(err)
		return nil, err
	}

	requesterNode := &RequesterNode{
		NodeID:    nodeId,
		Transport: transport,
	}

	transport.Subscribe(ctx, func(jobEvent *executor.JobEvent, job *executor.Job) {
		// we only care about jobs that we own
		if job.Owner != nodeId {
			return
		}

		switch jobEvent.EventName {

		// a compute node has bid on a job
		// let's decide if we want to accept it or not
		// we would call out to the reputation system
		// we also pay attention to the job deal concurrency setting
		case executor.JobEventBid:

			bidAccepted, message, err := requesterNode.ConsiderBid(job, jobEvent.NodeId)
			if err != nil {
				threadLogger.Warn().Msgf("There was an error considering bid: %s", err)
				return
			}

			if bidAccepted {
				logger.LogJobEvent(logger.JobEvent{
					Node: nodeId,
					Type: "requestor_node:bid_accepted",
					Job:  job.Id,
				})
				// TODO: Check result of accept job bid
				err = transport.AcceptJobBid(ctx, jobEvent.JobId, jobEvent.NodeId)
				if err != nil {
					threadLogger.Error().Err(err)
				}
			} else {
				logger.LogJobEvent(logger.JobEvent{
					Node: nodeId,
					Type: "requestor_node:bid_rejected",
					Job:  job.Id,
				})
				// TODO: Check result of reject job bid
				err = transport.RejectJobBid(ctx, jobEvent.JobId, jobEvent.NodeId, message)
				if err != nil {
					threadLogger.Error().Err(err)
				}
			}
		}
	})

	return requesterNode, nil
}

// a compute node has bid on the job
// should we accept the bid or not?
func (node *RequesterNode) ConsiderBid(job *executor.Job, nodeId string) (bool, string, error) {
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

	return true, "", nil
}

func (node *RequesterNode) newSpanForJob(ctx context.Context, jobID,
	name string) (context.Context, trace.Span) {

	return system.Span(ctx, "requestor_node/requester_node", name,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("nodeID", node.NodeID),
			attribute.String("jobID", jobID),
		),
	)
}

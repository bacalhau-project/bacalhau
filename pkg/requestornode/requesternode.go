package requestornode

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type RequesterNode struct {
	NodeID    string
	Transport transport.Transport
	Verifiers map[verifier.VerifierType]verifier.Verifier
}

func NewRequesterNode(
	cm *system.CleanupManager,
	t transport.Transport,
	verifiers map[verifier.VerifierType]verifier.Verifier,
) (*RequesterNode, error) {
	ctx := context.Background() // TODO: instrument with trace

	nodeID, err := t.HostID(ctx)
	threadLogger := logger.LoggerWithRuntimeInfo(nodeID)

	if err != nil {
		threadLogger.Error().Err(err)
		return nil, err
	}

	requesterNode := &RequesterNode{
		NodeID:    nodeID,
		Transport: t,
		Verifiers: verifiers,
	}

	t.Subscribe(ctx, func(ctx context.Context,
		jobEvent executor.JobEvent, job executor.Job) {
		// we only care about jobs that we own
		if job.Owner != nodeID {
			return
		}

		switch jobEvent.EventName { // nolint:gocritic // Switch statement will be used eventually
		// a compute node has bid on a job
		// let's decide if we want to accept it or not
		// we would call out to the reputation system
		// we also pay attention to the job deal concurrency setting
		case executor.JobEventBid:
			// Need to declare span separately to prevent shadowing
			var span trace.Span
			ctx, span = requesterNode.newSpanForJob(ctx,
				job.ID, "JobEventBid")
			defer span.End()

			bidAccepted, message, err := requesterNode.ConsiderBid(job, jobEvent.NodeID)
			if err != nil {
				threadLogger.Warn().Msgf("There was an error considering bid: %s", err)
				return
			}

			if bidAccepted {
				logger.LogJobEvent(logger.JobEvent{
					Node: nodeID,
					Type: "requestor_node:bid_accepted",
					Job:  job.ID,
				})
				// TODO: Check result of accept job bid
				err = t.AcceptJobBid(ctx, jobEvent.JobID, jobEvent.NodeID)
				if err != nil {
					threadLogger.Error().Err(err)
				}
			} else {
				logger.LogJobEvent(logger.JobEvent{
					Node: nodeID,
					Type: "requestor_node:bid_rejected",
					Job:  job.ID,
				})
				// TODO: Check result of reject job bid
				err = t.RejectJobBid(ctx, jobEvent.JobID, jobEvent.NodeID, message)
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
func (node *RequesterNode) ConsiderBid(job executor.Job, nodeID string) (bidAccepted bool, reason string, err error) {
	threadLogger := logger.LoggerWithNodeAndJobInfo(nodeID, job.ID)

	concurrency := job.Deal.Concurrency
	threadLogger.Debug().Msgf("Concurrency for this job: %d", concurrency)

	// we are already over-subscribed
	if len(job.Deal.AssignedNodes) >= concurrency {
		// nolint:lll // Error message needs long line
		threadLogger.Debug().Msgf("Rejected: Job already on enough nodes (Subscribed: %d vs Concurrency: %d", len(job.Deal.AssignedNodes), concurrency)
		return false, "Job is oversubscribed", nil
	}

	// sanity check to not allow a node to bid on a job twice
	alreadyAssigned := false

	for _, assignedNode := range job.Deal.AssignedNodes {
		if assignedNode == nodeID {
			alreadyAssigned = true
		}
	}

	if alreadyAssigned {
		return false, "This node is already assigned", nil
	}

	return true, "", nil
}

func (node *RequesterNode) PinContext(buildContext string) (string, error) {
	ipfsVerifier := node.Verifiers[verifier.VerifierIpfs]
	// TODO: we should have a method specifically for this not just piggybacking on the ipfs verifier
	return ipfsVerifier.ProcessResultsFolder(context.Background(), "", buildContext)
}

func (node *RequesterNode) newSpanForJob(ctx context.Context, jobID, name string) (context.Context, trace.Span) {
	return system.Span(ctx, "requestor_node/requester_node", name,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("nodeID", node.NodeID),
			attribute.String("jobID", jobID),
		),
	)
}

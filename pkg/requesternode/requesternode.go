package requesternode

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/controller"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type RequesterNodeConfig struct{}

type RequesterNode struct {
	id         string
	config     RequesterNodeConfig // nolint:gocritic
	controller *controller.Controller
	verifiers  map[verifier.VerifierType]verifier.Verifier
}

func NewRequesterNode(
	cm *system.CleanupManager,
	c *controller.Controller,
	verifiers map[verifier.VerifierType]verifier.Verifier,
	config RequesterNodeConfig, //nolint:gocritic
) (*RequesterNode, error) {
	// TODO: instrument with trace
	ctx := context.Background()
	nodeID, err := c.HostID(ctx)
	if err != nil {
		return nil, err
	}
	threadLogger := logger.LoggerWithRuntimeInfo(nodeID)
	if err != nil {
		threadLogger.Error().Err(err)
		return nil, err
	}
	requesterNode := &RequesterNode{
		id:         nodeID,
		config:     config,
		controller: c,
		verifiers:  verifiers,
	}

	requesterNode.subscriptionSetup()

	return requesterNode, nil
}

/*

  subscriptions

*/
func (node *RequesterNode) subscriptionSetup() {
	node.controller.Subscribe(func(ctx context.Context, jobEvent executor.JobEvent) {
		job, err := node.controller.GetJob(ctx, jobEvent.JobID)
		if err != nil {
			log.Error().Msgf("could not get job: %s - %s", jobEvent.JobID, err.Error())
			return
		}
		// we only care about jobs that we own
		if job.Data.RequesterNodeID != node.id {
			return
		}
		switch jobEvent.EventName {
		case executor.JobEventBid:
			node.subscriptionEventBid(ctx, jobEvent, job.Data)
		}
	})
}

func (node *RequesterNode) subscriptionEventBid(ctx context.Context, jobEvent executor.JobEvent, job executor.Job) {
	// Need to declare span separately to prevent shadowing
	var span trace.Span
	ctx, span = node.newSpanForJob(ctx, job.ID, "JobEventBid")
	defer span.End()

	threadLogger := logger.LoggerWithNodeAndJobInfo(node.id, job.ID)

	bidAccepted, _, err := node.considerBid(job, jobEvent.TargetNodeID)
	if err != nil {
		threadLogger.Warn().Msgf("There was an error considering bid: %s", err)
		return
	}

	if bidAccepted {
		logger.LogJobEvent(logger.JobEvent{
			Node: node.id,
			Type: "requestor_node:bid_accepted",
			Job:  job.ID,
		})
		err = node.controller.AcceptJobBid(ctx, jobEvent.JobID, jobEvent.TargetNodeID)
		if err != nil {
			threadLogger.Error().Err(err)
		}
	} else {
		logger.LogJobEvent(logger.JobEvent{
			Node: node.id,
			Type: "requestor_node:bid_rejected",
			Job:  job.ID,
		})
		err = node.controller.RejectJobBid(ctx, jobEvent.JobID, jobEvent.TargetNodeID)
		if err != nil {
			threadLogger.Error().Err(err)
		}
	}
}

// a compute node has bid on the job
// should we accept the bid or not?
func (node *RequesterNode) considerBid(job executor.Job, nodeID string) (bidAccepted bool, reason string, err error) {
	threadLogger := logger.LoggerWithNodeAndJobInfo(nodeID, job.ID)

	concurrency := job.Deal.Concurrency
	threadLogger.Debug().Msgf("Concurrency for this job: %d", concurrency)

	// we are already over-subscribed
	// if len(job.Deal.AssignedNodes) >= concurrency {
	// 	// nolint:lll // Error message needs long line
	// 	threadLogger.Debug().Msgf("Rejected: Job already on enough nodes (Subscribed: %d vs Concurrency: %d", len(job.Deal.AssignedNodes), concurrency)
	// 	return false, "Job is oversubscribed", nil
	// }

	// // sanity check to not allow a node to bid on a job twice
	// alreadyAssigned := false

	// for _, assignedNode := range job.Deal.AssignedNodes {
	// 	if assignedNode == nodeID {
	// 		alreadyAssigned = true
	// 	}
	// }

	// if alreadyAssigned {
	// 	return false, "This node is already assigned", nil
	// }

	return true, "", nil
}

func (node *RequesterNode) PinContext(buildContext string) (string, error) {
	ipfsVerifier := node.verifiers[verifier.VerifierIpfs]
	// TODO: we should have a method specifically for this not just piggybacking on the ipfs verifier
	return ipfsVerifier.ProcessResultsFolder(context.Background(), "", buildContext)
}

func (node *RequesterNode) newSpanForJob(ctx context.Context, jobID, name string) (context.Context, trace.Span) {
	return system.Span(ctx, "requestor_node/requester_node", name,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("nodeID", node.id),
			attribute.String("jobID", jobID),
		),
	)
}

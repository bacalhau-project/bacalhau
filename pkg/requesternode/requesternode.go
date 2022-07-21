package requesternode

import (
	"context"
	"sync"

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
	bidMutex   sync.Mutex
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
		if job.RequesterNodeID != node.id {
			return
		}
		switch jobEvent.EventName {
		case executor.JobEventBid:
			node.subscriptionEventBid(ctx, job, jobEvent)
		}
	})
}

func (node *RequesterNode) subscriptionEventBid(ctx context.Context, job executor.Job, jobEvent executor.JobEvent) {
	node.bidMutex.Lock()
	defer node.bidMutex.Unlock()

	// Need to declare span separately to prevent shadowing
	var span trace.Span
	ctx, span = node.newSpanForJob(ctx, job.ID, "JobEventBid")
	defer span.End()

	threadLogger := logger.LoggerWithNodeAndJobInfo(node.id, job.ID)

	accepted := func() bool {
		concurrency := job.Deal.Concurrency
		// let's see how many bids we have already accepted
		// it's important this comes from "local events"
		// otherwise we are in a race with the network and could
		// end up accepting many more bids than our concurrency
		localEvents, err := node.controller.GetJobLocalEvents(ctx, job.ID)
		if err != nil {
			threadLogger.Warn().Msgf("There was an error getting job events %s: %s", job.ID, err)
			return false
		}

		acceptedEvents := []executor.JobLocalEvent{}

		for _, localEvent := range localEvents {
			if localEvent.EventName == executor.JobLocalEventBidAccepted {
				// have we got a bid for a node we have already accepted a bid for?
				if localEvent.TargetNodeID == jobEvent.TargetNodeID {
					threadLogger.Debug().Msgf("Rejected: Job bid already accepted: %s %s", jobEvent.TargetNodeID, jobEvent.JobID)
					return false
				}
				acceptedEvents = append(acceptedEvents, localEvent)
			}
		}

		if len(acceptedEvents) >= concurrency {
			// nolint:lll // Error message needs long line
			threadLogger.Debug().Msgf("Rejected: Job already on enough nodes (Subscribed: %d vs Concurrency: %d", len(acceptedEvents), concurrency)
			return false
		}

		return true
	}()

	if accepted {
		logger.LogJobEvent(logger.JobEvent{
			Node: node.id,
			Type: "requestor_node:bid_accepted",
			Job:  job.ID,
		})
		err := node.controller.AcceptJobBid(ctx, jobEvent.JobID, jobEvent.SourceNodeID)
		if err != nil {
			threadLogger.Error().Err(err)
		}
	} else {
		logger.LogJobEvent(logger.JobEvent{
			Node: node.id,
			Type: "requestor_node:bid_rejected",
			Job:  job.ID,
		})
		err := node.controller.RejectJobBid(ctx, jobEvent.JobID, jobEvent.SourceNodeID)
		if err != nil {
			threadLogger.Error().Err(err)
		}
	}
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

package requesternode

import (
	"context"
	"fmt"
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
	id             string
	config         RequesterNodeConfig //nolint:gocritic
	controller     *controller.Controller
	verifiers      map[verifier.VerifierType]verifier.Verifier
	componentMutex sync.Mutex
	bidMutex       sync.Mutex
	verifyMutex    sync.Mutex
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
		case executor.JobEventResultsProposed:
			node.subscriptionEventShardExecutionComplete(ctx, job, jobEvent)
		case executor.JobEventError:
			node.subscriptionEventShardExecutionComplete(ctx, job, jobEvent)
		}
	})
}

func (node *RequesterNode) subscriptionEventBid(
	ctx context.Context,
	job executor.Job,
	jobEvent executor.JobEvent,
) {
	node.bidMutex.Lock()
	defer node.bidMutex.Unlock()

	// Need to declare span separately to prevent shadowing
	var span trace.Span
	ctx, span = node.newSpanForJob(ctx, job.ID, "JobEventBid")
	defer span.End()

	threadLogger := logger.LoggerWithNodeAndJobInfo(node.id, job.ID)

	accepted := func() bool {
		// let's see how many bids we have already accepted
		// it's important this comes from "local events"
		// otherwise we are in a race with the network and could
		// end up accepting many more bids than our concurrency
		localEvents, err := node.controller.GetJobLocalEvents(ctx, job.ID)
		if err != nil {
			threadLogger.Warn().Msgf("There was an error getting job events %s: %s", job.ID, err)
			return false
		}

		// a map of shard index onto an array of node ids we have farmed the job out to
		assignedNodes := map[int][]string{}

		for _, localEvent := range localEvents {
			if localEvent.EventName == executor.JobLocalEventBidAccepted {
				assignedNodesForShard, ok := assignedNodes[localEvent.ShardIndex]
				if !ok {
					assignedNodesForShard = []string{}
				}
				assignedNodesForShard = append(assignedNodesForShard, localEvent.TargetNodeID)
				assignedNodes[localEvent.ShardIndex] = assignedNodesForShard
			}
		}

		assignedNodesForShard, ok := assignedNodes[jobEvent.ShardIndex]
		if !ok {
			assignedNodesForShard = []string{}
		}

		// we have already reached concurrency for this shard
		// so let's reject this bid
		if len(assignedNodesForShard) >= job.Deal.Concurrency {
			//nolint:lll // Error message needs long line
			threadLogger.Debug().Msgf("Rejected: Job shard %s %d already reached concurrency of %d %+v", job.ID, jobEvent.ShardIndex, job.Deal.Concurrency, assignedNodesForShard)
			return false
		}

		return true
	}()

	if accepted {
		log.Debug().Msgf("Requester node %s accepting bid: %s %d", node.id, jobEvent.JobID, jobEvent.ShardIndex)
		err := node.controller.AcceptJobBid(ctx, jobEvent.JobID, jobEvent.SourceNodeID, jobEvent.ShardIndex)
		if err != nil {
			threadLogger.Error().Err(err)
		}
	} else {
		log.Debug().Msgf("Requester node %s rejecting bid: %s %d", node.id, jobEvent.JobID, jobEvent.ShardIndex)
		err := node.controller.RejectJobBid(ctx, jobEvent.JobID, jobEvent.SourceNodeID, jobEvent.ShardIndex)
		if err != nil {
			threadLogger.Error().Err(err)
		}
	}
}

// called for both JobEventShardCompleted and JobEventShardError
// we ask the verifier "IsExecutionComplete" to decide if we can start
// verifying the results - each verifier might have a different
// answer for IsExecutionComplete so we pass off to it to decide
// we mark the job as "verifying" to prevent duplicate verification
func (node *RequesterNode) subscriptionEventShardExecutionComplete(
	ctx context.Context,
	job executor.Job,
	jobEvent executor.JobEvent,
) {
	node.verifyMutex.Lock()
	defer node.verifyMutex.Unlock()
	err := node.shardExecutionComplete(ctx, job, jobEvent)
	if err != nil {
		err = node.controller.ShardError(
			ctx,
			job.ID,
			jobEvent.ShardIndex,
			err.Error(),
			[]byte{},
		)
		if err != nil {
			log.Debug().Msgf("ErrorShard failed: %s", err.Error())
		}
	}
}

func (node *RequesterNode) shardExecutionComplete(
	ctx context.Context,
	job executor.Job,
	jobEvent executor.JobEvent,
) error {
	threadLogger := logger.LoggerWithNodeAndJobInfo(node.id, job.ID)
	verifier, err := node.getVerifier(ctx, job.Spec.Verifier)
	if err != nil {
		return err
	}
	// ask the verifier if we have enough to start the verification yet
	isExecutionComplete, err := verifier.IsExecutionComplete(ctx, job.ID)
	if err != nil {
		return err
	}
	if !isExecutionComplete {
		return nil
	}
	// check that we have not already verified this job
	hasVerified, err := node.controller.HasLocalEvent(ctx, job.ID, controller.EventFilterByType(executor.JobLocalEventVerified))
	if err != nil {
		return err
	}
	if hasVerified {
		return nil
	}
	verificationResults, err := verifier.VerifyJob(ctx, job.ID)
	if err != nil {
		return err
	}
	// loop over each verification result and publish events
	for _, verificationResult := range verificationResults {
		if verificationResult.Verified {
			log.Debug().Msgf("Requester node %s accepting results: job=%s node=%s shard=%d", node.id, verificationResult.JobID, verificationResult.NodeID, jobEvent.ShardIndex)
			err := node.controller.AcceptResults(ctx, jobEvent.JobID, jobEvent.SourceNodeID, jobEvent.ShardIndex)
			if err != nil {
				threadLogger.Error().Err(err)
			}
		} else {
			log.Debug().Msgf("Requester node %s rejecting results: job=%s node=%s shard=%d", node.id, verificationResult.JobID, verificationResult.NodeID, jobEvent.ShardIndex)
			err := node.controller.RejectResults(ctx, jobEvent.JobID, jobEvent.SourceNodeID, jobEvent.ShardIndex)
			if err != nil {
				threadLogger.Error().Err(err)
			}
		}
	}
	return nil
}

//nolint:dupl // methods are not duplicates
func (node *RequesterNode) getVerifier(ctx context.Context, typ verifier.VerifierType) (verifier.Verifier, error) {
	node.componentMutex.Lock()
	defer node.componentMutex.Unlock()

	if _, ok := node.verifiers[typ]; !ok {
		return nil, fmt.Errorf(
			"no matching verifier found on this server: %s", typ.String())
	}

	v := node.verifiers[typ]
	installed, err := v.IsInstalled(ctx)
	if err != nil {
		return nil, err
	}
	if !installed {
		return nil, fmt.Errorf("verifier is not installed: %s", typ.String())
	}

	return v, nil
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

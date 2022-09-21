package requesternode

import (
	"context"
	"fmt"
	"time"

	sync "github.com/lukemarsden/golang-mutex-tracer"

	"github.com/filecoin-project/bacalhau/pkg/controller"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
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
	verifiers      map[model.VerifierType]verifier.Verifier
	componentMutex sync.Mutex
	bidMutex       sync.Mutex
	verifyMutex    sync.Mutex
}

func NewRequesterNode(
	ctx context.Context,
	cm *system.CleanupManager,
	c *controller.Controller,
	verifiers map[model.VerifierType]verifier.Verifier,
	config RequesterNodeConfig, //nolint:gocritic
) (*RequesterNode, error) {
	// TODO: instrument with trace
	nodeID := c.HostID()
	requesterNode := &RequesterNode{
		id:         nodeID,
		config:     config,
		controller: c,
		verifiers:  verifiers,
	}
	requesterNode.bidMutex.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "RequesterNode.bidMutex",
	})
	requesterNode.bidMutex.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "RequesterNode.bidMutex",
	})

	requesterNode.subscriptionSetup()

	return requesterNode, nil
}

/*
subscriptions
*/
func (node *RequesterNode) subscriptionSetup() {
	node.controller.Subscribe(func(ctx context.Context, jobEvent model.JobEvent) {
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
		case model.JobEventBid:
			node.subscriptionEventBid(ctx, job, jobEvent)
		case model.JobEventResultsProposed:
			node.subscriptionEventShardExecutionComplete(ctx, job, jobEvent)
		case model.JobEventError:
			node.subscriptionEventShardExecutionComplete(ctx, job, jobEvent)
		}
	})
}

func (node *RequesterNode) subscriptionEventBid(
	ctx context.Context,
	job model.Job,
	jobEvent model.JobEvent,
) {
	node.bidMutex.Lock()
	defer node.bidMutex.Unlock()

	// Need to declare span separately to prevent shadowing
	var span trace.Span
	ctx, span = node.newSpanForJob(ctx, job.ID, "JobEventBid")
	defer span.End()

	threadLogger := logger.LoggerWithNodeAndJobInfo(node.id, job.ID)
	bidQueueResults, err := processIncomingBid(ctx, node.controller, job, jobEvent)

	if err != nil {
		threadLogger.Warn().Msgf("There was an error calling processIncomingBid %s: %s", job.ID, err)
		return
	}

	for _, bidQueueResult := range bidQueueResults {
		if bidQueueResult.accepted {
			log.Debug().Msgf("Requester node %s accepting bid: %s %d", node.id, job.ID, jobEvent.ShardIndex)
			err := node.controller.AcceptJobBid(ctx, job.ID, bidQueueResult.nodeID, jobEvent.ShardIndex)
			if err != nil {
				threadLogger.Error().Err(err)
			}
		} else {
			log.Debug().Msgf("Requester node %s rejecting bid: %s %d", node.id, job.ID, jobEvent.ShardIndex)
			err := node.controller.RejectJobBid(ctx, job.ID, bidQueueResult.nodeID, jobEvent.ShardIndex)
			if err != nil {
				threadLogger.Error().Err(err)
			}
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
	job model.Job,
	jobEvent model.JobEvent,
) {
	node.verifyMutex.Lock()
	defer node.verifyMutex.Unlock()
	err := node.attemptVerification(ctx, job)
	if err != nil {
		err = node.controller.ShardError(
			ctx,
			job.ID,
			jobEvent.ShardIndex,
			err.Error(),
			nil,
		)
		if err != nil {
			log.Debug().Msgf("ErrorShard failed: %s", err.Error())
		}
	}
}

func (node *RequesterNode) attemptVerification(
	ctx context.Context,
	job model.Job,
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
	hasVerified, err := node.controller.HasLocalEvent(ctx, job.ID, controller.EventFilterByType(model.JobLocalEventVerified))
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
			log.Debug().Msgf(
				"Requester node %s accepting results: job=%s node=%s shard=%d",
				node.id,
				verificationResult.JobID,
				verificationResult.NodeID,
				verificationResult.ShardIndex,
			)
			err := node.controller.AcceptResults(ctx, verificationResult.JobID, verificationResult.NodeID, verificationResult.ShardIndex)
			if err != nil {
				threadLogger.Error().Err(err)
			}
		} else {
			log.Debug().Msgf(
				"Requester node %s rejecting results: job=%s node=%s shard=%d",
				node.id,
				verificationResult.JobID,
				verificationResult.NodeID,
				verificationResult.ShardIndex,
			)
			err := node.controller.RejectResults(ctx, verificationResult.JobID, verificationResult.NodeID, verificationResult.ShardIndex)
			if err != nil {
				threadLogger.Error().Err(err)
			}
		}
	}
	return node.controller.CompleteVerification(ctx, job.ID)
}

//nolint:dupl // methods are not duplicates
func (node *RequesterNode) getVerifier(ctx context.Context, typ model.VerifierType) (verifier.Verifier, error) {
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

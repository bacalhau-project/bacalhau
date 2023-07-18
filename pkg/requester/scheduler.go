package requester

import (
	"context"
	"net/url"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/secrets"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util"
	sync "github.com/bacalhau-project/golang-mutex-tracer"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type BaseSchedulerParams struct {
	ID                string
	Host              host.Host
	JobStore          jobstore.Store
	NodeSelector      NodeSelector
	RetryStrategy     RetryStrategy
	ComputeEndpoint   compute.Endpoint
	StorageProviders  storage.StorageProvider
	EventEmitter      EventEmitter
	GetVerifyCallback func() *url.URL
}

type BaseScheduler struct {
	id                string
	host              host.Host
	jobStore          jobstore.Store
	nodeSelector      NodeSelector
	retryStrategy     RetryStrategy
	computeService    compute.Endpoint
	storageProviders  storage.StorageProvider
	eventEmitter      EventEmitter
	getVerifyCallback func() *url.URL
	mu                sync.Mutex
}

func NewBaseScheduler(params BaseSchedulerParams) *BaseScheduler {
	res := &BaseScheduler{
		id:                params.ID,
		host:              params.Host,
		jobStore:          params.JobStore,
		nodeSelector:      params.NodeSelector,
		retryStrategy:     params.RetryStrategy,
		computeService:    params.ComputeEndpoint,
		storageProviders:  params.StorageProviders,
		eventEmitter:      params.EventEmitter,
		getVerifyCallback: params.GetVerifyCallback,
	}

	// TODO: replace with job level lock
	res.mu.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "Scheduler.mu",
	})
	return res
}

func (s *BaseScheduler) StartJob(ctx context.Context, req StartJobRequest) (err error) {
	defer func() {
		if err != nil {
			s.stopJob(ctx, req.Job.ID(), err.Error(), false)
		}
	}()

	// find nodes that can execute the job
	selectedNodes, err := s.nodeSelector.SelectNodes(ctx, &req.Job)
	if err != nil {
		return err
	}

	err = s.jobStore.UpdateJobState(ctx, jobstore.UpdateJobStateRequest{
		JobID: req.Job.Metadata.ID,
		Condition: jobstore.UpdateJobCondition{
			ExpectedState: model.JobStateNew,
		},
		NewState: model.JobStateInProgress,
	})
	if err != nil {
		return err
	}
	s.eventEmitter.EmitJobCreated(ctx, req.Job)

	go s.notifyAskForBid(logger.ContextWithNodeIDLogger(context.Background(), s.id), trace.LinkFromContext(ctx), req.Job, selectedNodes)
	return err
}

func (s *BaseScheduler) CancelJob(ctx context.Context, request CancelJobRequest) (CancelJobResult, error) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s received CancelJob for job: %s with reason %s",
		s.id, request.JobID, request.Reason)

	jobState, err := s.jobStore.GetJobState(ctx, request.JobID)
	if err != nil {
		return CancelJobResult{}, err
	}
	if jobState.State.IsTerminal() {
		return CancelJobResult{}, NewErrJobAlreadyTerminal(request.JobID)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopJob(ctx, jobState.JobID, request.Reason, request.UserTriggered)
	return CancelJobResult{}, nil
}

//////////////////////////////
//   Compute Proxy Methods  //
//////////////////////////////

func (s *BaseScheduler) notifyAskForBid(ctx context.Context, link trace.Link, job model.Job, nodes []model.NodeInfo) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester.Scheduler.StartJob",
		trace.WithLinks(link), // link to any api traces
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String(model.TracerAttributeNameNodeID, s.id),
			attribute.String(model.TracerAttributeNameJobID, job.Metadata.ID),
		),
	)
	defer span.End()

	// persist the intent to ask the node for a bid, which is helpful to avoid asking an unresponsive node again during retries.
	// we persist the intent for all nodes before asking any node to bid, so that we don't fail the job if the first node we ask rejects the
	// the bid before we persist the intent to ask the other nodes.
	for _, node := range nodes {
		executionID := model.ExecutionID{
			JobID:       job.Metadata.ID,
			NodeID:      node.PeerInfo.ID.String(),
			ExecutionID: "e-" + uuid.NewString(),
		}

		err := s.jobStore.CreateExecution(ctx, model.ExecutionState{
			JobID:            executionID.JobID,
			NodeID:           executionID.NodeID,
			ComputeReference: executionID.ExecutionID,
			State:            model.ExecutionStateAskForBid,
		})
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("error creating execution")
			return
		}

		// TODO: Move this ...
		// We need to decode the environment for any env vars where the value begins ENC[hex-encoded-var]
		if job.Spec.Engine == model.EngineDocker {
			if len(job.Spec.Docker.EnvironmentVariables) > 0 {
				newEnv, err := secrets.RecryptEnvString(job.Spec.Docker.EnvironmentVariables,
					s.host.Peerstore().PrivKey(s.host.ID()),     // Us (requester node)
					s.host.Peerstore().PubKey(node.PeerInfo.ID), // Compute node's public key
				)
				if err != nil {
					log.Ctx(ctx).Error().Err(err).Msg("failed to re-encrypt secrets")
				}
				job.Spec.Docker.EnvironmentVariables = newEnv
			}
		}

		newCtx := util.NewDetachedContext(ctx)
		go s.doNotifyAskForBid(newCtx, job, executionID)
	}
}

func (s *BaseScheduler) doNotifyAskForBid(ctx context.Context, job model.Job, executionID model.ExecutionID) {
	request := compute.AskForBidRequest{
		Job: job,
		ExecutionMetadata: compute.ExecutionMetadata{
			ExecutionID: executionID.ExecutionID,
			JobID:       executionID.JobID,
		},
		RoutingMetadata: compute.RoutingMetadata{
			SourcePeerID: s.id,
			TargetPeerID: executionID.NodeID,
		},
	}
	_, err := s.computeService.AskForBid(ctx, request)
	if err != nil {
		s.handleExecutionFailure(ctx, executionID, err)
	}
}

func (s *BaseScheduler) updateAndNotifyBidAccepted(ctx context.Context, execution model.ExecutionState) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s responding with BidAccepted for bid: %s", s.id, execution.ComputeReference)
	err := s.jobStore.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
		ExecutionID: execution.ID(),
		Condition: jobstore.UpdateExecutionCondition{
			ExpectedState:   execution.State,
			ExpectedVersion: execution.Version,
		},
		NewValues: model.ExecutionState{
			State: model.ExecutionStateBidAccepted,
		},
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to update execution state to BidAccepted. %s", execution)
	} else {
		go func(ctx context.Context) {
			request := compute.BidAcceptedRequest{
				ExecutionID: execution.ComputeReference,
				RoutingMetadata: compute.RoutingMetadata{
					SourcePeerID: s.id,
					TargetPeerID: execution.NodeID,
				},
			}
			response, notifyErr := s.computeService.BidAccepted(ctx, request)
			if notifyErr != nil {
				s.handleExecutionFailure(ctx, execution.ID(), notifyErr)
			} else {
				s.eventEmitter.EmitBidAccepted(ctx, request, response)
			}
		}(util.NewDetachedContext(ctx))
	}
}

func (s *BaseScheduler) updateAndNotifyBidRejected(ctx context.Context, execution model.ExecutionState) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s responding with BidRejected for bid: %s", s.id, execution.ComputeReference)
	err := s.jobStore.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
		ExecutionID: execution.ID(),
		Condition: jobstore.UpdateExecutionCondition{
			ExpectedState:   execution.State,
			ExpectedVersion: execution.Version,
		},
		NewValues: model.ExecutionState{
			State: model.ExecutionStateBidRejected,
		},
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to update execution state to BidRejected. %s", execution)
	} else {
		go func(ctx context.Context) {
			request := compute.BidRejectedRequest{
				ExecutionID: execution.ComputeReference,
				RoutingMetadata: compute.RoutingMetadata{
					SourcePeerID: s.id,
					TargetPeerID: execution.NodeID,
				},
			}
			response, notifyErr := s.computeService.BidRejected(ctx, request)
			if notifyErr != nil {
				log.Ctx(ctx).Error().Err(notifyErr).Msgf("failed to notify BidRejected for bid: %s", execution.ComputeReference)
			}
			s.eventEmitter.EmitBidRejected(ctx, request, response)
		}(util.NewDetachedContext(ctx))
	}
}

// notifyCancel only notifies compute nodes and doesn't update the execution state in the job store. This is because
// the execution state is updated when stopping/failing the job itself.
func (s *BaseScheduler) notifyCancel(ctx context.Context, message string, execution model.ExecutionState) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s responding with Cancel for bid: %s", s.id, execution.ComputeReference)

	go func(ctx context.Context) {
		request := compute.CancelExecutionRequest{
			ExecutionID:   execution.ComputeReference,
			Justification: message,
			RoutingMetadata: compute.RoutingMetadata{
				SourcePeerID: s.id,
				TargetPeerID: execution.NodeID,
			},
		}
		_, err := s.computeService.CancelExecution(ctx, request)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("failed to notify cancellation for execution: %s", execution.ComputeReference)
		}
	}(util.NewDetachedContext(ctx))
}

// /////////////////////////////
// Compute callback handlers //
// /////////////////////////////

// OnBidComplete implements compute.Callback
func (s *BaseScheduler) OnBidComplete(ctx context.Context, response compute.BidResult) {
	log.Ctx(ctx).Debug().Msgf("Requester node received bid response %+v", response)

	executionID := model.ExecutionID{
		JobID:       response.JobID,
		NodeID:      response.SourcePeerID,
		ExecutionID: response.ExecutionID,
	}

	newState := model.ExecutionStateAskForBidRejected
	if response.Accepted {
		newState = model.ExecutionStateAskForBidAccepted
	}
	err := s.jobStore.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
		ExecutionID: executionID,
		Condition: jobstore.UpdateExecutionCondition{
			ExpectedState: model.ExecutionStateAskForBid,
		},
		NewValues: model.ExecutionState{
			State:  newState,
			Status: response.Reason,
		},
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[OnBidComplete] failed to update execution")
		return
	}

	if response.Accepted {
		s.eventEmitter.EmitBidReceived(ctx, response)
	}
	s.TransitionJobState(ctx, executionID.JobID)
}

func (s *BaseScheduler) OnRunComplete(ctx context.Context, result compute.RunResult) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s received RunComplete for execution: %s from %s",
		s.id, result.ExecutionID, result.SourcePeerID)
	s.eventEmitter.EmitRunComplete(ctx, result)

	// update execution state
	err := s.jobStore.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
		ExecutionID: model.ExecutionID{
			JobID:       result.JobID,
			NodeID:      result.SourcePeerID,
			ExecutionID: result.ExecutionID,
		},
		Condition: jobstore.UpdateExecutionCondition{
			ExpectedState: model.ExecutionStateBidAccepted,
		},
		NewValues: model.ExecutionState{
			PublishedResult: result.PublishResult,
			RunOutput:       result.RunCommandResult,
			State:           model.ExecutionStateCompleted,
		},
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[OnRunComplete] failed to update execution")
		return
	}

	s.TransitionJobState(ctx, result.JobID)
}

func (s *BaseScheduler) OnCancelComplete(ctx context.Context, result compute.CancelResult) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s received CancelComplete for execution: %s from %s",
		s.id, result.ExecutionID, result.SourcePeerID)
}

func (s *BaseScheduler) OnComputeFailure(ctx context.Context, result compute.ComputeError) {
	log.Ctx(ctx).Debug().Err(result).Msgf("Requester node %s received ComputeFailure for execution: %s from %s",
		s.id, result.ExecutionID, result.SourcePeerID)
	s.handleExecutionFailure(ctx, model.ExecutionID{
		JobID:       result.JobID,
		NodeID:      result.SourcePeerID,
		ExecutionID: result.ExecutionID,
	}, result)
}

func (s *BaseScheduler) handleExecutionFailure(ctx context.Context, executionID model.ExecutionID, failure error) {
	// update execution state
	err := s.jobStore.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
		ExecutionID: executionID,
		Condition: jobstore.UpdateExecutionCondition{
			UnexpectedStates: []model.ExecutionStateType{
				model.ExecutionStateCompleted,
				model.ExecutionStateCancelled,
			},
		},
		NewValues: model.ExecutionState{
			State:  model.ExecutionStateFailed,
			Status: failure.Error(),
		},
		Comment: failure.Error(),
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[handleExecutionFailure] failed to update execution")
		return
	}

	s.eventEmitter.EmitComputeFailure(ctx, executionID, failure)
	s.TransitionJobState(ctx, executionID.JobID)
}

// make sure to call this function with the lock held
func (s *BaseScheduler) stopJob(ctx context.Context, jobID, reason string, userRequested bool) {
	if userRequested {
		log.Ctx(ctx).Info().Msgf("stopping job %s because the user requested it", jobID)
	} else {
		log.Ctx(ctx).Error().Err(errors.New(reason)).Msgf("error completing job %s", jobID)
	}

	cancelledExecutions, err := jobstore.StopJob(ctx, s.jobStore, jobID, reason, userRequested)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[stopJob] failed to stop job")
	}

	for _, execution := range cancelledExecutions {
		s.notifyCancel(ctx, reason, execution)
	}
	eventName := model.JobEventError
	if userRequested {
		eventName = model.JobEventCanceled
	}
	s.eventEmitter.EmitEventSilently(ctx, model.JobEvent{
		SourceNodeID: s.id,
		JobID:        jobID,
		Status:       reason,
		EventName:    eventName,
		EventTime:    time.Now(),
	})
}

// compile-time check that BackendCallback implements the expected interfaces
var _ Scheduler = (*BaseScheduler)(nil)
var _ compute.Callback = (*BaseScheduler)(nil)

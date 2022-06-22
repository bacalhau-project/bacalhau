package compute_node

import (
	"context"
	"fmt"
	"sync"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type ComputeNode struct {
	NodeID             string
	Mutex              sync.Mutex
	Transport          transport.Transport
	Executors          map[executor.EngineType]executor.Executor
	Verifiers          map[verifier.VerifierType]verifier.Verifier
	JobSelectionPolicy JobSelectionPolicy
}

func NewComputeNode(
	transport transport.Transport,
	executors map[executor.EngineType]executor.Executor,
	verifiers map[verifier.VerifierType]verifier.Verifier,
	jobSelectionPolicy JobSelectionPolicy,
) (*ComputeNode, error) {
	ctx := context.Background()
	nodeID, err := transport.HostID(ctx)
	if err != nil {
		return nil, err
	}

	computeNode := &ComputeNode{
		NodeID:             nodeID,
		Transport:          transport,
		Verifiers:          verifiers,
		Executors:          executors,
		JobSelectionPolicy: jobSelectionPolicy,
	}

	transport.Subscribe(ctx, func(ctx context.Context,
		jobEvent *executor.JobEvent, job *executor.Job) {

		switch jobEvent.EventName {
		// a new job has arrived - decide if we want to bid on it
		case executor.JobEventCreated:
			ctx, span := computeNode.newSpanForJob(ctx,
				job.Id, "JobEventCreated")
			defer span.End()

			shouldRun, err := computeNode.SelectJob(ctx,
				JobSelectionPolicyProbeData{
					NodeId: nodeID,
					JobId:  jobEvent.JobId,
					Spec:   jobEvent.JobSpec,
				})
			if err != nil {
				log.Error().Msgf("error checking job policy: %v", err)
				return
			}

			if shouldRun {
				// TODO: Why do we have two different kinds of loggers?
				logger.LogJobEvent(logger.JobEvent{
					Node: nodeID,
					Type: "compute_node:bid",
					Job:  job.Id,
				})

				log.Debug().Msgf("compute node %s bidding on: %+v", nodeID,
					jobEvent.JobSpec)

				// TODO: Check result of bid job
				err = transport.BidJob(ctx, jobEvent.JobId)
				if err != nil {
					log.Error().Msgf("error bidding on job: %v", err)
				}
				return
			} else {
				log.Debug().Msgf("compute node %s skipped bidding on: %+v",
					nodeID, jobEvent.JobSpec)
			}

		// we have been given the goahead to run the job
		case executor.JobEventBidAccepted:
			// we only care if the accepted bid is for us
			if jobEvent.NodeId != nodeID {
				return
			}

			ctx, span := computeNode.newSpanForJob(ctx,
				job.Id, "JobEventBidAccepted")
			defer span.End()

			log.Debug().Msgf("Bid accepted: Server (id: %s) - Job (id: %s)", nodeID, job.Id)

			logger.LogJobEvent(logger.JobEvent{
				Node: nodeID,
				Type: "compute_node:run",
				Job:  job.Id,
				Data: job,
			})

			resultFolder, err := computeNode.RunJob(ctx, job)
			if err != nil {
				log.Error().Msgf("Error running the job: %s %+v", err, job)
				_ = transport.ErrorJob(ctx, job.Id, fmt.Sprintf("Error running the job: %s", err))
				return
			}

			verifier, err := computeNode.getVerifier(ctx, job.Spec.Verifier)
			if err != nil {
				log.Error().Msgf("Error geting the verifier for the job: %s %+v", err, job)
				_ = transport.ErrorJob(ctx, job.Id, fmt.Sprintf("Error geting the verifier for the job: %s", err))
				return
			}

			resultValue, err := verifier.ProcessResultsFolder(
				ctx, job.Id, resultFolder)
			if err != nil {
				log.Error().Msgf("Error verifying results: %s %+v", err, job)
				_ = transport.ErrorJob(ctx, job.Id, fmt.Sprintf("Error verifying results: %s", err))
				return
			}

			logger.LogJobEvent(logger.JobEvent{
				Node: nodeID,
				Type: "compute_node:result",
				Job:  job.Id,
				Data: resultValue,
			})

			if err = transport.SubmitResult(
				ctx,
				job.Id,
				fmt.Sprintf("Got job result: %s", resultValue),
				resultValue,
			); err != nil {
				log.Error().Msgf("Error submitting result: %s %+v", err, job)
				_ = transport.ErrorJob(ctx, job.Id, fmt.Sprintf("Error running the job: %s", err))
				return
			}
		}
	})

	return computeNode, nil
}

// how this is implemented could be improved
// for example - it should be possible to shell out to a user-defined program or send a HTTP request
// with the detauils of the job (input CIDs, submitter reputation etc)
// that will decide if it's worth doing the job or not
// for now - the rule is "do we have all the input CIDS"
// TODO: allow user probes (http / exec) to be used to decide if we should run the job
func (node *ComputeNode) SelectJob(
	ctx context.Context,
	data JobSelectionPolicyProbeData,
) (bool, error) {

	// check that we have the executor and it's installed
	executor, err := node.getExecutor(ctx, data.Spec.Engine)
	if err != nil {
		return false, err
	}

	// check that we have the verifier and it's installed
	_, err = node.getVerifier(ctx, data.Spec.Verifier)
	if err != nil {
		return false, err
	}

	return ApplyJobSelectionPolicy(
		ctx,
		node.JobSelectionPolicy,
		executor,
		data,
	)
}

func (node *ComputeNode) RunJob(ctx context.Context, job *executor.Job) (
	string, error) {

	// check that we have the executor to run this job
	executor, err := node.getExecutor(ctx, job.Spec.Engine)
	if err != nil {
		return "", err
	}

	return executor.RunJob(ctx, job)
}

func (node *ComputeNode) getExecutor(ctx context.Context,
	typ executor.EngineType) (executor.Executor, error) {

	node.Mutex.Lock()
	defer node.Mutex.Unlock()

	if _, ok := node.Executors[typ]; !ok {
		return nil, fmt.Errorf(
			"no matching executor found on this server: %s", typ.String())
	}

	executorEngine := node.Executors[typ]
	installed, err := executorEngine.IsInstalled(ctx)
	if err != nil {
		return nil, err
	}
	if !installed {
		return nil, fmt.Errorf("executor is not installed: %s", typ.String())
	}

	return executorEngine, nil
}

func (node *ComputeNode) getVerifier(ctx context.Context,
	typ verifier.VerifierType) (verifier.Verifier, error) {

	node.Mutex.Lock()
	defer node.Mutex.Unlock()

	if _, ok := node.Verifiers[typ]; !ok {
		return nil, fmt.Errorf(
			"no matching verifier found on this server: %s", typ.String())
	}

	verifier := node.Verifiers[typ]
	installed, err := verifier.IsInstalled(ctx)
	if err != nil {
		return nil, err
	}
	if !installed {
		return nil, fmt.Errorf("verifier is not installed: %s", typ.String())
	}

	return verifier, nil
}

func (node *ComputeNode) newSpanForJob(ctx context.Context, jobID,
	name string) (context.Context, trace.Span) {

	return system.Span(ctx, "compute_node/compute_node", name,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("nodeID", node.NodeID),
			attribute.String("jobID", jobID),
		),
	)
}

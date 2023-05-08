package requester

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
)

type startJobHandler func(context.Context, StartJobRequest) error
type cancelJobHandler func(context.Context, CancelJobRequest) (CancelJobResult, error)
type verifyExecutionsHandler func(context.Context, []verifier.VerifierResult) (succeeded, failed []verifier.VerifierResult)

var (
	successfulStartJobHandler startJobHandler = func(ctx context.Context, sjr StartJobRequest) error {
		return nil
	}
	successfulCancelJobHandler cancelJobHandler = func(ctx context.Context, cjr CancelJobRequest) (CancelJobResult, error) {
		return CancelJobResult{}, nil
	}
)

type mockScheduler struct {
	handleStartJob         startJobHandler
	handleCancelJob        cancelJobHandler
	handleVerifyExecutions verifyExecutionsHandler
}

// OnCancelComplete implements Scheduler
func (*mockScheduler) OnCancelComplete(ctx context.Context, result compute.CancelResult) {
	panic("unimplemented")
}

// OnComputeFailure implements Scheduler
func (*mockScheduler) OnComputeFailure(ctx context.Context, err compute.ComputeError) {
	panic("unimplemented")
}

// OnPublishComplete implements Scheduler
func (*mockScheduler) OnPublishComplete(ctx context.Context, result compute.PublishResult) {
	panic("unimplemented")
}

// OnRunComplete implements Scheduler
func (*mockScheduler) OnRunComplete(ctx context.Context, result compute.RunResult) {
	panic("unimplemented")
}

// CancelJob implements Scheduler
func (m *mockScheduler) CancelJob(ctx context.Context, cjr CancelJobRequest) (CancelJobResult, error) {
	if m.handleCancelJob == nil {
		m.handleCancelJob = successfulCancelJobHandler
	}
	return m.handleCancelJob(ctx, cjr)
}

// StartJob implements Scheduler
func (m *mockScheduler) StartJob(ctx context.Context, sjr StartJobRequest) error {
	if m.handleStartJob == nil {
		m.handleStartJob = successfulStartJobHandler
	}
	return m.handleStartJob(ctx, sjr)
}

// TransitionJobState implements Scheduler
func (m *mockScheduler) VerifyExecutions(
	ctx context.Context,
	results []verifier.VerifierResult,
) (succeeded, failed []verifier.VerifierResult) {
	if m.handleVerifyExecutions != nil {
		return m.handleVerifyExecutions(ctx, results)
	}
	return results, nil
}

var _ Scheduler = (*mockScheduler)(nil)

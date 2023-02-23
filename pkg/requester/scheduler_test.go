package requester

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/compute"
)

type startJobHandler func(context.Context, StartJobRequest) error
type cancelJobHandler func(context.Context, CancelJobRequest) (CancelJobResult, error)

var (
	successfulStartJobHandler startJobHandler = func(ctx context.Context, sjr StartJobRequest) error {
		return nil
	}
	successfulCancelJobHandler cancelJobHandler = func(ctx context.Context, cjr CancelJobRequest) (CancelJobResult, error) {
		return CancelJobResult{}, nil
	}
)

type mockScheduler struct {
	handleStartJob  startJobHandler
	handleCancelJob cancelJobHandler
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

var _ Scheduler = (*mockScheduler)(nil)

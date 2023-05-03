package moderation

import (
	"context"

	"github.com/bacalhau-project/bacalhau/dashboard/api/pkg/types"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
)

type combinedModerator struct {
	execModerator   ExecutionModerator
	resultModerator ResultsModerator
	manualModerator ManualModerator
}

func NewCombinedModerator(e ExecutionModerator, r ResultsModerator, m ManualModerator) ManualModerator {
	return &combinedModerator{execModerator: e, resultModerator: r, manualModerator: m}
}

// ShouldExecute implements ManualModerator
func (m *combinedModerator) ShouldExecute(ctx context.Context,
	probe *bidstrategy.JobSelectionPolicyProbeData,
) (*bidstrategy.BidStrategyResponse, error) {
	return m.execModerator.ShouldExecute(ctx, probe)
}

// Verify implements ManualModerator
func (m *combinedModerator) Verify(ctx context.Context, req verifier.VerifierRequest) ([]verifier.VerifierResult, error) {
	return m.resultModerator.Verify(ctx, req)
}

// Moderate implements ManualModerator
func (m *combinedModerator) Moderate(ctx context.Context,
	requestID int64,
	approved bool,
	reason string,
	user *types.User,
) (types.Moderation, error) {
	return m.manualModerator.Moderate(ctx, requestID, approved, reason, user)
}

var _ ManualModerator = (*combinedModerator)(nil)

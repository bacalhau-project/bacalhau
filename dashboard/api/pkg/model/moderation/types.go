package moderation

import (
	"context"

	"github.com/bacalhau-project/bacalhau/dashboard/api/pkg/types"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
)

type ExecutionModerator interface {
	// ShouldExecute returns a response signaling whether the passed job should
	// be executed, queued or canceled. If the initial response is queue,
	// subsequent calls may return execute or cancel after which the value
	// should no longer change.
	ShouldExecute(context.Context, *bidstrategy.JobSelectionPolicyProbeData) (*bidstrategy.BidStrategyResponse, error)
}

type ResultsModerator interface {
	// Verify returns the subset of passed executions that should be
	// verified or rejected. The return value may not include all of the passed
	// executions; any missing are pending a decision.
	Verify(context.Context, verifier.VerifierRequest) ([]verifier.VerifierResult, error)
}

type ManualModerator interface {
	ExecutionModerator
	ResultsModerator

	// Moderate specifies a decision for a manual moderation request.
	Moderate(ctx context.Context, requestID int64, approved bool, reason string, user *types.User) (types.Moderation, error)
}

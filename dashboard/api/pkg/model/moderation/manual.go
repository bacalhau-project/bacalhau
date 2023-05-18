package moderation

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/bacalhau-project/bacalhau/dashboard/api/pkg/store"
	"github.com/bacalhau-project/bacalhau/dashboard/api/pkg/types"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/v1beta1"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
	"go.uber.org/multierr"
	"golang.org/x/exp/slices"
)

// Response returned to signal that a job will be moderated.
var waitResponse = bidstrategy.BidStrategyResponse{
	ShouldWait: true,
	Reason:     "Awaiting human approval",
}

type manualModerator struct {
	store *store.PostgresStore
}

func NewManualModerator(store *store.PostgresStore) ManualModerator {
	return &manualModerator{store: store}
}

// ShouldExecute implements ExecutionModerator
func (m *manualModerator) ShouldExecute(
	ctx context.Context,
	probe *bidstrategy.JobSelectionPolicyProbeData,
) (*bidstrategy.BidStrategyResponse, error) {
	// Do we have an approval for the job? If so, return it.
	resp, err := m.store.GetJobModerations(ctx, probe.JobID)
	if err != nil {
		return nil, err
	}

	idx := slices.IndexFunc(resp, func(moderation types.JobModerationSummary) bool {
		return moderation.Request.GetType() == types.ModerationTypeExecution
	})
	if idx >= 0 {
		return asBidStrategyResponse(resp[idx].Moderation), nil
	}

	// No approval found. Is there an approval request?
	requests, err := m.store.GetModerationRequestsForJobOfType(ctx, probe.JobID, types.ModerationTypeExecution)
	if err == nil && len(requests) == 0 {
		// We don't have a request, so create one.
		// TODO: we should probably reject requests for jobs that are already running.
		_, err = m.store.CreateJobModerationRequest(ctx, probe.JobID, types.ModerationTypeExecution, types.ParseURLPtr(probe.Callback))
	}

	// There is an open request – we are just waiting for it to be handled.
	return &waitResponse, err
}

// Verify implements ResultsModerator
func (m *manualModerator) Verify(ctx context.Context, req verifier.VerifierRequest) ([]verifier.VerifierResult, error) {
	storedRequests, err := m.store.GetModerationRequestsForJobOfType(ctx, req.JobID, types.ModerationTypeResult)
	if err != nil {
		return nil, err
	}

	requests := make(map[model.ExecutionID]types.ResultModerationRequest, len(req.Executions))
	for _, request := range storedRequests {
		if resultRequest, ok := request.(*types.ResultModerationRequest); ok {
			requests[resultRequest.ExecutionID] = *resultRequest
		}
	}

	// For any executions that don't have a request, we need to create one.
	for _, execution := range req.Executions {
		if _, found := requests[execution.ID()]; found {
			continue
		}

		cerr := m.createRequestForExecution(ctx, execution, req.Callback)
		err = multierr.Append(err, cerr)
	}
	if err != nil {
		return nil, err
	}

	storedModerations, err := m.store.GetJobModerations(ctx, req.JobID)
	if err != nil {
		return nil, err
	}

	// Now convert the list of moderations into verification results.
	results := make([]verifier.VerifierResult, 0, len(req.Executions))
	for _, moderation := range storedModerations {
		if resultRequest, ok := moderation.Request.(*types.ResultModerationRequest); ok {
			results = append(results, verifier.VerifierResult{
				ExecutionID: resultRequest.ExecutionID,
				Verified:    moderation.Moderation.Status,
			})
		}
	}

	return results, err
}

func (m *manualModerator) createRequestForExecution(ctx context.Context, execution model.ExecutionState, callback *url.URL) error {
	var executionSpec v1beta1.StorageSpec
	err := json.Unmarshal(execution.VerificationProposal, &executionSpec)
	if err != nil {
		return err
	}

	return m.store.CreateResultModerationRequest(ctx, execution.ID(), executionSpec, types.ParseURLPtr(callback))
}

// Moderate implements ResultsModerator
func (m *manualModerator) Moderate(ctx context.Context,
	requestID int64,
	approved bool,
	reason string,
	user *types.User,
) (types.Moderation, error) {
	moderation := types.Moderation{
		RequestID:     requestID,
		UserAccountID: user.ID,
		Status:        approved,
		Notes:         reason,
	}

	return moderation, m.store.CreateJobModeration(ctx, moderation)
}

var _ ManualModerator = (*manualModerator)(nil)

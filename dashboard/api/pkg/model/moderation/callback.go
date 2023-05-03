package moderation

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bacalhau-project/bacalhau/dashboard/api/pkg/store"
	"github.com/bacalhau-project/bacalhau/dashboard/api/pkg/types"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
	"github.com/bacalhau-project/bacalhau/pkg/verifier/external"
	"github.com/pkg/errors"
)

type callbackModerator struct {
	store      *store.PostgresStore
	underlying ManualModerator
}

func NewCallbackModerator(store *store.PostgresStore, underlying ManualModerator) ManualModerator {
	return &callbackModerator{store: store, underlying: underlying}
}

// ShouldExecute implements ManualModerator
func (callback *callbackModerator) ShouldExecute(
	ctx context.Context,
	probe *bidstrategy.JobSelectionPolicyProbeData,
) (*bidstrategy.BidStrategyResponse, error) {
	return callback.underlying.ShouldExecute(ctx, probe)
}

// Verify implements ManualModerator
func (callback *callbackModerator) Verify(ctx context.Context, req verifier.VerifierRequest) ([]verifier.VerifierResult, error) {
	return callback.underlying.Verify(ctx, req)
}

// Moderate implements ManualModerator
func (callback *callbackModerator) Moderate(ctx context.Context,
	requestID int64,
	approved bool,
	reason string,
	user *types.User,
) (types.Moderation, error) {
	moderation, err := callback.underlying.Moderate(ctx, requestID, approved, reason, user)
	if err != nil {
		return moderation, err
	}

	request, err := callback.store.GetModerationRequest(ctx, requestID)
	if err != nil || request.GetCallback() == nil || !request.GetCallback().IsAbs() {
		// No callback associated with this – end early.
		return moderation, err
	}

	mismatchedTypesErr := fmt.Errorf("programming error: mismatched %q request of type %T", request.GetType(), request)

	var callbackData any
	switch request.GetType() {
	case types.ModerationTypeExecution:
		if realRequest, ok := request.(*types.JobModerationRequest); ok {
			callbackData = bidstrategy.ModerateJobRequest{
				ClientID: system.GetClientID(),
				JobID:    realRequest.JobID,
				Response: *asBidStrategyResponse(&moderation),
			}
		} else {
			return moderation, mismatchedTypesErr
		}
	case types.ModerationTypeResult:
		if realRequest, ok := request.(*types.ResultModerationRequest); ok {
			callbackData = external.ExternalVerificationResponse{
				ClientID: system.GetClientID(),
				Verifications: []verifier.VerifierResult{
					{ExecutionID: realRequest.ExecutionID, Verified: approved},
				},
			}
		} else {
			return moderation, mismatchedTypesErr
		}
	default:
		return moderation, nil
	}

	port, err := strconv.ParseUint(request.GetCallback().Port(), 10, 16)
	if err != nil {
		return moderation, err
	}

	client := publicapi.NewAPIClient(request.GetCallback().Hostname(), uint16(port))
	err = client.PostSigned(ctx, request.GetCallback().RequestURI(), callbackData, nil)
	return moderation, errors.Wrap(err, "response from callback")
}

var _ ManualModerator = (*callbackModerator)(nil)

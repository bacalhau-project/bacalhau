package publicapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/handlerwrapper"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type cancelRequest = SignedRequest[model.JobCancelPayload]

type cancelResponse struct {
	State *model.JobState `json:"state"`
}

// cancel godoc
//
//	@ID						pkg/requester/publicapi/cancel
//	@Summary				Cancels the job with the job-id specified in the body payload.
//	@Description.markdown	endpoints_cancel
//	@Tags					Job
//	@Accept					json
//	@Produce				json
//	@Param					cancelRequest	body		cancelRequest	true	" "
//	@Success				200				{object}	cancelResponse
//	@Failure				400				{object}	string
//	@Failure				401				{object}	string
//	@Failure				403				{object}	string
//	@Failure				500				{object}	string
//	@Router					/requester/cancel [post]
func (s *RequesterAPIServer) cancel(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	jobCancelPayload, err := unmarshalSignedJob[model.JobCancelPayload](ctx, req.Body)
	if err != nil {
		httpError(ctx, res, err, http.StatusBadRequest)
		return
	}

	res.Header().Set(handlerwrapper.HTTPHeaderClientID, jobCancelPayload.ClientID)
	ctx = system.AddJobIDToBaggage(ctx, jobCancelPayload.ClientID)

	// Get the job, check it exists and check it belongs to the same client
	job, err := s.jobStore.GetJob(ctx, jobCancelPayload.JobID)
	if err != nil {
		log.Ctx(ctx).Debug().Msgf("Missing job: %s", err)
		http.Error(res, bacerrors.ErrorToErrorResponse(err), http.StatusBadRequest)
		return
	}

	// We can compare the payload's client ID against the existing job's metadata
	// as we have confirmed the public key that the request was signed with matches
	// the client ID the request claims.
	if job.Metadata.ClientID != jobCancelPayload.ClientID {
		log.Ctx(ctx).Debug().Msgf("Mismatched ClientIDs for cancel, existing job: %s and cancel request: %s",
			job.Metadata.ClientID, jobCancelPayload.ClientID)

		errorResponse := bacerrors.ErrorToErrorResponse(errors.Errorf("mismatched client id: %s", jobCancelPayload.ClientID))
		http.Error(res, errorResponse, http.StatusForbidden)
		return
	}

	_, err = s.requester.CancelJob(ctx, requester.CancelJobRequest{
		JobID:         jobCancelPayload.JobID,
		Reason:        jobCancelPayload.Reason,
		UserTriggered: true,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	jobState, err := getJobStateFromJobID(ctx, s, jobCancelPayload.JobID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Header().Set(handlerwrapper.HTTPHeaderClientID, jobCancelPayload.ClientID)
	res.Header().Set(handlerwrapper.HTTPHeaderJobID, jobCancelPayload.JobID)
	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(cancelResponse{
		State: &jobState,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getJobStateFromJobID(ctx context.Context, apiServer *RequesterAPIServer, jobID string) (model.JobState, error) {
	return apiServer.jobStore.GetJobState(ctx, jobID)
}

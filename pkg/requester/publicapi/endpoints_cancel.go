package publicapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/handlerwrapper"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type cancelRequest = publicapi.SignedRequest[model.JobCancelPayload] //nolint:unused // Swagger wants this

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
	jobCancelPayload, err := publicapi.UnmarshalSigned[model.JobCancelPayload](ctx, req.Body)
	if err != nil {
		publicapi.HTTPError(ctx, res, err, http.StatusBadRequest)
		return
	}

	res.Header().Set(handlerwrapper.HTTPHeaderClientID, jobCancelPayload.ClientID)
	ctx = system.AddJobIDToBaggage(ctx, jobCancelPayload.ClientID)

	// Get the job, check it exists and check it belongs to the same client
	job, err := s.jobStore.GetJob(ctx, jobCancelPayload.JobID)
	if err != nil {
		publicapi.HTTPError(ctx, res, errors.Wrap(err, "missing job"), http.StatusNotFound)
		return
	}

	// We can compare the payload's client ID against the existing job's metadata
	// as we have confirmed the public key that the request was signed with matches
	// the client ID the request claims.
	if job.Metadata.ClientID != jobCancelPayload.ClientID {
		err = fmt.Errorf("mismatched ClientIDs for cancel, existing job: %s and cancel request: %s",
			job.Metadata.ClientID, jobCancelPayload.ClientID)
		publicapi.HTTPError(ctx, res, err, http.StatusUnauthorized)
		return
	}

	_, err = s.requester.CancelJob(ctx, requester.CancelJobRequest{
		JobID:         jobCancelPayload.JobID,
		Reason:        jobCancelPayload.Reason,
		UserTriggered: true,
	})
	if err != nil {
		publicapi.HTTPError(ctx, res, err, http.StatusInternalServerError)
		return
	}

	jobState, err := getJobStateFromJobID(ctx, s, jobCancelPayload.JobID)
	if err != nil {
		publicapi.HTTPError(ctx, res, err, http.StatusInternalServerError)
		return
	}

	res.Header().Set(handlerwrapper.HTTPHeaderClientID, jobCancelPayload.ClientID)
	res.Header().Set(handlerwrapper.HTTPHeaderJobID, jobCancelPayload.JobID)
	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(cancelResponse{
		State: &jobState,
	})
	if err != nil {
		publicapi.HTTPError(ctx, res, err, http.StatusInternalServerError)
		return
	}
}

func getJobStateFromJobID(ctx context.Context, apiServer *RequesterAPIServer, jobID string) (model.JobState, error) {
	return apiServer.jobStore.GetJobState(ctx, jobID)
}

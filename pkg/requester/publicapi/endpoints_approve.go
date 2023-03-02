package publicapi

import (
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/rs/zerolog/log"
)

type JobApprovePayload struct {
	requester.ApproveJobRequest
}

func (j JobApprovePayload) GetClientID() string {
	return j.ApproveJobRequest.ClientID
}

func (s *RequesterAPIServer) approve(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	approval, err := unmarshalSignedJob[JobApprovePayload](ctx, req.Body)
	if err != nil {
		httpError(ctx, res, err, http.StatusBadRequest)
		return
	}

	ctx = log.Ctx(ctx).With().Str("JobID", approval.ApproveJobRequest.JobID).Logger().WithContext(ctx)
	err = s.requester.ApproveJob(ctx, approval.ApproveJobRequest)
	if err != nil {
		httpError(ctx, res, err, http.StatusBadRequest)
		return
	}
}

package publicapi

import (
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/rs/zerolog/log"
)

func (s *RequesterAPIServer) approve(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	approval, err := publicapi.UnmarshalSigned[bidstrategy.ModerateJobRequest](ctx, req.Body)
	if err != nil {
		publicapi.HTTPError(ctx, res, err, http.StatusBadRequest)
		return
	}

	ctx = log.Ctx(ctx).With().Str("JobID", approval.JobID).Logger().WithContext(ctx)
	err = s.requester.ApproveJob(ctx, approval)
	if err != nil {
		publicapi.HTTPError(ctx, res, err, http.StatusBadRequest)
		return
	}
}

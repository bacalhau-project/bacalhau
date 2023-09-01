package requester

import (
	"context"
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models/migration/legacy"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/go-chi/render"
	"github.com/rs/zerolog/log"
)

// list godoc
//
//	@ID						pkg/requester/publicapi/list
//	@Summary				Simply lists jobs.
//	@Description.markdown	endpoints_list
//	@Tags					Job
//	@Accept					json
//	@Produce				json
//	@Param					ListRequest	body		apimodels.ListRequest	true	"Set `return_all` to `true` to return all jobs on the network (may degrade performance, use with care!)."
//	@Success				200			{object}	apimodels.ListResponse
//	@Failure				400			{object}	string
//	@Failure				500			{object}	string
//	@Router					/api/v1/requester/list [post]
//
//nolint:lll
func (s *Endpoint) list(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	var listReq apimodels.ListRequest
	if err := render.DecodeJSON(req.Body, &listReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	res.Header().Set(apimodels.HTTPHeaderClientID, listReq.ClientID)
	res.Header().Set(apimodels.HTTPHeaderJobID, listReq.JobID)

	jobList, err := s.getJobsList(ctx, listReq)
	if err != nil {
		_, ok := err.(*bacerrors.JobNotFound)
		if ok {
			http.Error(res, bacerrors.ErrorToErrorResponse(err), http.StatusBadRequest)
			return
		}
	}

	jobWithInfos := make([]*model.JobWithInfo, len(jobList))
	for i, job := range jobList {
		jobState, innerErr := legacy.GetJobState(ctx, s.jobStore, job.ID())
		if innerErr != nil {
			log.Ctx(ctx).Error().Err(innerErr).Msg("error getting job states")
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		jobWithInfos[i] = &model.JobWithInfo{
			Job:   job,
			State: jobState,
		}
	}

	render.JSON(res, req, apimodels.ListResponse{Jobs: jobWithInfos})
}

func (s *Endpoint) getJobsList(ctx context.Context, listReq apimodels.ListRequest) ([]model.Job, error) {
	list, err := s.jobStore.GetJobs(ctx, jobstore.JobQuery{
		Namespace:   listReq.ClientID,
		ID:          listReq.JobID,
		Limit:       listReq.MaxJobs,
		IncludeTags: listReq.IncludeTags,
		ExcludeTags: listReq.ExcludeTags,
		ReturnAll:   listReq.ReturnAll,
		SortBy:      listReq.SortBy,
		SortReverse: listReq.SortReverse,
	})
	if err != nil {
		return nil, err
	}
	res := make([]model.Job, len(list))
	for i := range list {
		legacyJob, err := legacy.ToLegacyJob(&list[i])
		if err != nil {
			return nil, err
		}
		res[i] = *legacyJob
	}
	return res, nil
}

package requester

import (
	"context"
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models/migration/legacy"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels/legacymodels"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// @ID				pkg/requester/publicapi/list
// @Summary		Simply lists jobs.
// @Description	Returns the first (sorted) #`max_jobs` jobs that belong to the `client_id` passed in the body payload (by default).
// @Description	If `return_all` is set to true, it returns all jobs on the Bacalhau network.
// @Description	If `id` is set, it returns only the job with that ID.
// @Tags			Job
// @Accept			json
// @Produce		json
// @Param			ListRequest	body		legacymodels.ListRequest	true	"Set `return_all` to `true` to return all jobs on the network (may degrade performance, use with care!)."
// @Success		200			{object}	legacymodels.ListResponse
// @Failure		400			{object}	string
// @Failure		500			{object}	string
// @Router			/api/v1/requester/list [post]
//
//nolint:lll
func (s *Endpoint) list(c echo.Context) error {
	var listReq legacymodels.ListRequest
	var err error
	var jobList []model.Job

	if err := c.Bind(&listReq); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if listReq.JobID != "" {
		jobList, err = s.getSingleJob(c.Request().Context(), listReq.JobID)
	} else {
		jobList, err = s.getJobsList(c.Request().Context(), listReq)
	}

	if err != nil {
		_, ok := err.(*bacerrors.JobNotFound)
		if ok {
			http.Error(c.Response(), bacerrors.ErrorToErrorResponse(err), http.StatusBadRequest)
			return nil
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	jobWithInfos := make([]*model.JobWithInfo, len(jobList))
	for i, job := range jobList {
		jobState, innerErr := legacy.GetJobState(c.Request().Context(), s.jobStore, job.ID())
		if innerErr != nil {
			log.Ctx(c.Request().Context()).Error().Err(innerErr).Msg("error getting job states")
			return echo.NewHTTPError(http.StatusInternalServerError, innerErr.Error())
		}
		jobWithInfos[i] = &model.JobWithInfo{
			Job:   job,
			State: jobState,
		}
	}

	return c.JSON(http.StatusOK, legacymodels.ListResponse{
		Jobs: jobWithInfos,
	})
}

func (s *Endpoint) getSingleJob(ctx context.Context, jobID string) ([]model.Job, error) {
	job, err := s.jobStore.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	convertedJob, err := legacy.ToLegacyJob(&job)
	if err != nil {
		return nil, err
	}

	return []model.Job{*convertedJob}, nil
}

func (s *Endpoint) getJobsList(ctx context.Context, listReq legacymodels.ListRequest) ([]model.Job, error) {
	response, err := s.jobStore.GetJobs(ctx, jobstore.JobQuery{
		Namespace:   listReq.ClientID,
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
	res := make([]model.Job, len(response.Jobs))
	for i := range response.Jobs {
		legacyJob, err := legacy.ToLegacyJob(&response.Jobs[i])
		if err != nil {
			return nil, err
		}
		res[i] = *legacyJob
	}
	return res, nil
}

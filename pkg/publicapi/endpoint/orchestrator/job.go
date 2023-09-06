package orchestrator

import (
	"net/http"
	"strconv"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/labstack/echo/v4"
	"k8s.io/apimachinery/pkg/labels"
)

func (e *Endpoint) putJob(c echo.Context) error {
	ctx := c.Request().Context()
	var args apimodels.PutJobRequest
	if err := c.Bind(&args); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(&args); err != nil {
		return err
	}
	resp, err := e.orchestrator.SubmitJob(ctx, &orchestrator.SubmitJobRequest{
		Job: args.Job,
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, apimodels.PutJobResponse{
		JobID:        resp.JobID,
		EvaluationID: resp.EvaluationID,
	})
}

func (e *Endpoint) getJob(c echo.Context) error {
	ctx := c.Request().Context()
	jobID := c.Param("id")
	job, err := e.store.GetJob(ctx, jobID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, apimodels.GetJobResponse{
		Job: &job,
	})
}

func (e *Endpoint) listJobs(c echo.Context) error {
	ctx := c.Request().Context()
	var args apimodels.ListJobsRequest
	if err := c.Bind(&args); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(&args); err != nil {
		return err
	}

	var offset uint64
	var err error
	if args.NextToken != "" {
		offset, err = strconv.ParseUint(args.NextToken, 10, 32)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
	}

	// TODO: implement label selectors in jobstore instead of filtering here
	selector, err := parseLabels(c)
	if err != nil {
		return err
	}

	query := jobstore.JobQuery{
		Namespace:   args.Namespace,
		Limit:       args.Limit,
		Offset:      uint32(offset),
		SortBy:      args.OrderBy,
		SortReverse: args.Reverse,
	}

	if args.Namespace == apimodels.AllNamespacesNamespace {
		query.Namespace = ""
		query.ReturnAll = true
	}

	jobs, err := e.store.GetJobs(ctx, query)
	if err != nil {
		return err
	}

	res := &apimodels.ListJobsResponse{
		Jobs: make([]*models.Job, len(jobs)),
	}
	for i := range jobs {
		if selector.Matches(labels.Set(jobs[i].Labels)) {
			res.Jobs[i] = &jobs[i]
		}
	}
	return c.JSON(http.StatusOK, res)
}

func (e *Endpoint) stopJob(c echo.Context) error {
	ctx := c.Request().Context()
	var args apimodels.StopJobRequest
	if err := c.Bind(&args); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(&args); err != nil {
		return err
	}
	resp, err := e.orchestrator.StopJob(ctx, &orchestrator.StopJobRequest{
		JobID:         args.JobID,
		Reason:        args.Reason,
		UserTriggered: true,
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, &apimodels.StopJobResponse{
		EvaluationID: resp.EvaluationID,
	})
}

func (e *Endpoint) jobHistory(c echo.Context) error {
	ctx := c.Request().Context()
	jobID := c.Param("id")
	var args apimodels.ListJobHistoryRequest
	if err := c.Bind(&args); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(&args); err != nil {
		return err
	}

	options := jobstore.JobHistoryFilterOptions{
		Since:                 args.Since,
		ExcludeExecutionLevel: args.Type == "job",
		ExcludeJobLevel:       args.Type == "execution",
	}
	history, err := e.store.GetJobHistory(ctx, jobID, options)
	if err != nil {
		return err
	}
	res := &apimodels.ListJobHistoryResponse{
		History: make([]*models.JobHistory, len(history)),
	}
	for i := range history {
		res.History[i] = &history[i]
	}

	return c.JSON(http.StatusOK, res)
}

func (e *Endpoint) jobExecutions(c echo.Context) error {
	ctx := c.Request().Context()
	jobID := c.Param("id")
	var args apimodels.ListJobExecutionsRequest
	if err := c.Bind(&args); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(&args); err != nil {
		return err
	}

	executions, err := e.store.GetExecutions(ctx, jobID)
	if err != nil {
		return err
	}
	res := &apimodels.ListJobExecutionsResponse{
		Executions: make([]*models.Execution, len(executions)),
	}
	for i := range executions {
		res.Executions[i] = &executions[i]
	}
	return c.JSON(http.StatusOK, res)
}

func (e *Endpoint) jobResults(c echo.Context) error {
	ctx := c.Request().Context()
	jobID := c.Param("id")
	var args apimodels.ListJobResultsRequest
	if err := c.Bind(&args); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(&args); err != nil {
		return err
	}

	executions, err := e.store.GetExecutions(ctx, jobID)
	if err != nil {
		return err
	}
	results := make([]*models.SpecConfig, 0)
	for _, execution := range executions {
		if execution.ComputeState.StateType == models.ExecutionStateCompleted {
			results = append(results, execution.PublishedResult)
		}
	}
	return c.JSON(http.StatusOK, &apimodels.ListJobResultsResponse{
		Results: results,
	})
}

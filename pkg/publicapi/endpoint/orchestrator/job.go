package orchestrator

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"golang.org/x/exp/slices"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
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
		Warnings:     resp.Warnings,
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
		Jobs: make([]*models.Job, 0),
	}
	for i := range jobs {
		if selector.Matches(labels.Set(jobs[i].Labels)) {
			res.Jobs = append(res.Jobs, &jobs[i])
		}
	}
	return c.JSON(http.StatusOK, res)
}

func (e *Endpoint) stopJob(c echo.Context) error {
	ctx := c.Request().Context()
	jobID := c.Param("id")

	var args apimodels.StopJobRequest
	if err := c.Bind(&args); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(&args); err != nil {
		return err
	}
	resp, err := e.orchestrator.StopJob(ctx, &orchestrator.StopJobRequest{
		JobID:         jobID,
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
		ExcludeExecutionLevel: args.EventType == "job",
		ExcludeJobLevel:       args.EventType == "execution",
		ExecutionID:           args.ExecutionID,
		NodeID:                args.NodeID,
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

	// TODO: move ordering to jobstore
	// parse order_by
	var sortFnc func(a, b models.Execution) bool
	switch args.OrderBy {
	case "modify_time", "":
		sortFnc = func(a, b models.Execution) bool { return a.ModifyTime < b.ModifyTime }
	case "create_time":
		sortFnc = func(a, b models.Execution) bool { return a.CreateTime < b.CreateTime }
	case "id":
		sortFnc = func(a, b models.Execution) bool { return a.ID < b.ID }
	case "state":
		sortFnc = func(a, b models.Execution) bool { return a.ComputeState.StateType < b.ComputeState.StateType }
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid order_by")
	}
	if args.Reverse {
		baseSortFnc := sortFnc
		sortFnc = func(a, b models.Execution) bool { return !baseSortFnc(a, b) }
	}

	// query executions
	executions, err := e.store.GetExecutions(ctx, jobID)
	if err != nil {
		return err
	}

	// sort executions
	slices.SortFunc(executions, sortFnc)

	// apply limit
	if args.Limit > 0 && len(executions) > int(args.Limit) {
		executions = executions[:args.Limit]
	}

	// prepare result
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

	resp, err := e.orchestrator.GetResults(ctx, &orchestrator.GetResultsRequest{
		JobID: jobID,
	})
	if err != nil {
		return err
	}
	return publicapi.UnescapedJSON(c, http.StatusOK, &apimodels.ListJobResultsResponse{
		Results: resp.Results,
	})
}

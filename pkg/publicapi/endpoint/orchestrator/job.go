package orchestrator

import (
	"fmt"
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
)

// godoc for Orchestrator PutJob
//
// @ID			orchestrator/putJob
// @Summary		Submits a job to the orchestrator.
// @Description	Submits a job to the orchestrator.
// @Tags			Orchestrator
// @Accept		json
// @Produce		json
// @Param			job	body	models.Job	true	"Job to submit"
// @Success		200	{object}	apimodels.PutJobResponse
// @Failure		400	{object}	string
// @Failure		500	{object}	string
// @Router			/api/v1/orchestrator/jobs [put]
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

// godoc for Orchestrator GetJob
//
// @ID			orchestrator/getJob
// @Summary		Returns a job.
// @Description	Returns a job.
// @Tags			Orchestrator
// @Accept		json
// @Produce		json
// @Param			id	query	string	false	"ID to get the job for"
// @Success		200	{object}	apimodels.GetJobResponse
// @Failure		400	{object}	string
// @Failure		500	{object}	string
// @Router			/api/v1/orchestrator/jobs [get]
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

// godoc for Orchestrator ListJobs
//
// @ID			orchestrator/listJobs
// @Summary		Returns a list of jobs.
// @Description	Returns a list of jobs.
// @Tags			Orchestrator
// @Accept		json
// @Produce		json
// @Param			namespace	query	string	false	"Namespace to get the jobs for"
// @Param			limit	query	int	false			"Limit the number of jobs returned"
// @Param			next_token	query	string	false	"Token to get the next page of jobs"
// @Param			reverse	query	bool	false		"Reverse the order of the jobs"
// @Param			order_by	query	string	false	"Order the jobs by the given field"
// @Success		200	{object}	apimodels.ListJobsResponse
// @Failure		400	{object}	string
// @Failure		500	{object}	string
// @Router			/api/v1/orchestrator/jobs [get]
func (e *Endpoint) listJobs(c echo.Context) error {
	ctx := c.Request().Context()
	var args apimodels.ListJobsRequest
	if err := c.Bind(&args); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(&args); err != nil {
		return err
	}

	var offset uint32
	var err error

	// If the request contains a paging token then it is decoded and used to replace
	// any other values provided in the request. This allows for stable sorting to
	// allow the pagination to work correctly.
	if args.NextToken != "" {
		token, err := models.NewPagingTokenFromString(args.NextToken)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		// Overwrite any provided values with the ones from the token.
		args.OrderBy = token.SortBy
		args.Reverse = token.SortReverse
		args.Limit = token.Limit
		offset = token.Offset
	}

	selector, err := parseLabels(c)
	if err != nil {
		return err
	}

	query := jobstore.JobQuery{
		Namespace:   args.Namespace,
		Limit:       args.Limit,
		Offset:      offset,
		SortBy:      args.OrderBy,
		SortReverse: args.Reverse,
		Selector:    selector,
	}

	if args.Namespace == apimodels.AllNamespacesNamespace {
		query.Namespace = ""
		query.ReturnAll = true
	}

	response, err := e.store.GetJobs(ctx, query)
	if err != nil {
		return err
	}

	var nextToken string
	// If the next offset > 0 then it means there are more records to be returned, so
	// we should give the user a token to use that will return the next page of results.
	// We encode the current settings into the token to maintain a stable sort across
	// pages.
	if response.NextOffset != 0 {
		nextToken = models.NewPagingToken(&models.PagingTokenParams{
			SortBy:      args.OrderBy,
			SortReverse: args.Reverse,
			Limit:       args.Limit,
			Offset:      response.NextOffset,
		}).String()
	}

	res := &apimodels.ListJobsResponse{
		Jobs: lo.Map[models.Job, *models.Job](response.Jobs, func(item models.Job, _ int) *models.Job {
			return &item
		}),
		BaseListResponse: apimodels.BaseListResponse{
			NextToken: nextToken,
		},
	}

	return c.JSON(http.StatusOK, res)
}

// godoc for Orchestrator StopJob
//
// @ID			orchestrator/stopJob
// @Summary		Stops a job.
// @Description	Stops a job.
// @Tags			Orchestrator
// @Accept		json
// @Produce		json
// @Param			id	path	string	true	"ID to stop the job for"
// @Param			reason	query	string	false	"Reason for stopping the job"
// @Success		200	{object}	apimodels.StopJobResponse
// @Failure		400	{object}	string
// @Failure		500	{object}	string
// @Router			/api/v1/orchestrator/jobs/{id} [delete]
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

// godoc for Orchestrator JobHistory
//
// @ID			orchestrator/jobHistory
// @Summary		Returns the history of a job.
// @Description	Returns the history of a job.
// @Tags			Orchestrator
// @Accept		json
// @Produce		json
// @Param			id	path	string	true	"ID to get the job history for"
// @Param			since	query	string	false	"Only return history since this time"
// @Param			event_type	query	string	false	"Only return history of this event type"
// @Param			execution_id	query	string	false	"Only return history of this execution ID"
// @Param			node_id	query	string	false	"Only return history of this node ID"
// @Success		200	{object}	apimodels.ListJobHistoryResponse
// @Failure		400	{object}	string
// @Failure		500	{object}	string
// @Router			/api/v1/orchestrator/jobs/{id}/history [get]
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

// godoc for Orchestrator JobExecutions
//
// @ID			orchestrator/jobExecutions
// @Summary		Returns the executions of a job.
// @Description	Returns the executions of a job.
// @Tags			Orchestrator
// @Accept		json
// @Produce		json
// @Param			id	path	string	true	"ID to get the job executions for"
// @Param			limit	query	int	false			"Limit the number of executions returned"
// @Param			next_token	query	string	false	"Token to get the next page of executions"
// @Param			reverse	query	bool	false		"Reverse the order of the executions"
// @Param			order_by	query	string	false	"Order the executions by the given field"
// @Success		200	{object}	apimodels.ListJobExecutionsResponse
// @Failure		400	{object}	string
// @Failure		500	{object}	string
// @Router			/api/v1/orchestrator/jobs/{id}/executions [get]
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

// godoc for Orchestrator JobResults
//
// @ID			orchestrator/jobResults
// @Summary		Returns the results of a job.
// @Description	Returns the results of a job.
// @Tags			Orchestrator
// @Accept		json
// @Produce		json
// @Param			id	path	string	true	"ID to get the job results for"
// @Param			limit	query	int	false			"Limit the number of results returned"
// @Param			next_token	query	string	false	"Token to get the next page of results"
// @Param			reverse	query	bool	false		"Reverse the order of the results"
// @Param			order_by	query	string	false	"Order the results by the given field"
// @Success		200	{object}	apimodels.ListJobResultsResponse
// @Failure		400	{object}	string
// @Failure		500	{object}	string
// @Router			/api/v1/orchestrator/jobs/{id}/results [get]
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

// godoc for Orchestrator JobLogs
//
// @ID				orchestrator/logs
// @Summary			Displays the logs for a current job/execution
// @Description		Shows the output from the job specified by `id`
// @Description		The output will be continuous until either, the client disconnects or the execution completes.
// @Tags			Orchestrator
// @Accept			json
// @Produce			json
// @Param			id				path	string	true	"ID to get the job logs for"
// @Param			execution_id	query 	string	false	"Fetch logs for a specific execution"
// @Param			tail	query	bool	false	"Fetch historical logs"
// @Param			follow			query	bool	false	"Follow the logs"
// @Success		200			{object}	string
// @Failure		400			{object}	string
// @Failure		500			{object}	string
// @Router			/api/v1/orchestrator/jobs/{id}/logs [get]
func (e *Endpoint) logs(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return fmt.Errorf("failed to upgrade websocket connection: %w", err)
	}
	defer ws.Close()

	err = e.logsWS(c, ws)
	if err != nil {
		err = ws.WriteJSON(concurrency.AsyncResult[models.ExecutionLog]{
			Err: err,
		})
		if err != nil {
			c.Logger().Errorf("failed to write error to websocket: %s", err)
		}
	}
	_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	return nil
}

func (e *Endpoint) logsWS(c echo.Context, ws *websocket.Conn) error {
	jobID := c.Param("id")
	var args apimodels.GetLogsRequest
	if err := c.Bind(&args); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(&args); err != nil {
		return err
	}

	logstreamCh, err := e.orchestrator.ReadLogs(c.Request().Context(), orchestrator.ReadLogsRequest{
		JobID:       jobID,
		ExecutionID: args.ExecutionID,
		Tail:        args.Tail,
		Follow:      args.Follow,
	})
	if err != nil {
		return fmt.Errorf("failed to open log stream for job %s: %w", jobID, err)
	}

	for logMsg := range logstreamCh {
		if err = ws.WriteJSON(logMsg); err != nil {
			return err
		}
	}
	return nil
}

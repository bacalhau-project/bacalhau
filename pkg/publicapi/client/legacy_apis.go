package client

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	legacy_job "github.com/bacalhau-project/bacalhau/pkg/legacyjob"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels/legacymodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/signatures"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

// APIRetryCount - for some queries (like read events and read state)
// we want to fail early (10 seconds should be ample time)
// but retry a number of times - this is to avoid network
// flakes failing the canary
const APIRetryCount = 5
const APIShortTimeoutSeconds = 10

// List returns the list of jobs in the node's transport.
func (apiClient *APIClient) List(
	ctx context.Context,
	idFilter string,
	includeTags []string,
	excludeTags []string,
	maxJobs int,
	returnAll bool,
	sortBy string,
	sortReverse bool,
) (
	[]*model.JobWithInfo, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester/publicapi.APIClient.List")
	defer span.End()

	req := legacymodels.ListRequest{
		ClientID:    apiClient.ClientID,
		MaxJobs:     uint32(maxJobs),
		JobID:       idFilter,
		IncludeTags: includeTags,
		ExcludeTags: excludeTags,
		ReturnAll:   returnAll,
		SortBy:      sortBy,
		SortReverse: sortReverse,
	}

	var res legacymodels.ListResponse
	if err := apiClient.DoPost(ctx, "/api/v1/requester/list", req, &res); err != nil {
		return nil, err
	}

	return res.Jobs, nil
}

func (apiClient *APIClient) Nodes(ctx context.Context) ([]models.NodeInfo, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester/publicapi.APIClient.Nodes")
	defer span.End()

	var nodes []models.NodeInfo
	err := apiClient.doGet(ctx, "/api/v1/requester/nodes", &nodes)
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

// Cancel will request that the job with the specified ID is stopped. The JobInfo will be returned if the cancel
// was submitted. If no match is found, Cancel returns false with a nil error.
func (apiClient *APIClient) Cancel(ctx context.Context, jobID string, reason string) (*model.JobState, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester/publicapi.APIClient.Cancel")
	defer span.End()

	if jobID == "" {
		return &model.JobState{}, fmt.Errorf("jobID must be non-empty in a Cancel call")
	}

	// Check the existence of a job with the provided ID, whether it is a short or long ID.
	jobInfo, found, err := apiClient.Get(ctx, jobID)
	if err != nil {
		return &model.JobState{}, err
	}
	if !found {
		return &model.JobState{}, bacerrors.NewJobNotFound(jobID)
	}

	// We potentially used the short jobID which `Get` supports and so let's switch
	// to use the longer version.
	jobID = jobInfo.State.JobID

	// Create a payload before signing it with our local key (for verification on the
	// server).
	req := model.JobCancelPayload{
		ClientID: apiClient.ClientID,
		JobID:    jobID,
		Reason:   reason,
	}

	var res legacymodels.CancelResponse
	if err := apiClient.doPostSigned(ctx, "/api/v1/requester/cancel", req, &res); err != nil {
		return &model.JobState{}, err
	}

	return res.State, nil
}

// Get returns job data for a particular job ID. If no match is found, Get returns false with a nil error.
func (apiClient *APIClient) Get(ctx context.Context, jobID string) (*model.JobWithInfo, bool, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester/publicapi.APIClient.Get")
	defer span.End()

	if jobID == "" {
		return &model.JobWithInfo{}, false, fmt.Errorf("jobID must be non-empty in a Get call")
	}

	jobsList, err := apiClient.List(ctx, jobID, model.IncludeAny, model.ExcludeNone, 1, false, "created_at", true)
	if err != nil {
		return &model.JobWithInfo{}, false, err
	}

	if len(jobsList) > 0 {
		return jobsList[0], true, nil
	} else {
		return &model.JobWithInfo{}, false, bacerrors.NewJobNotFound(jobID)
	}
}

func (apiClient *APIClient) GetJobState(ctx context.Context, jobID string) (model.JobState, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester/publicapi.APIClient.GetJobState")
	defer span.End()

	if jobID == "" {
		return model.JobState{}, fmt.Errorf("jobID must be non-empty in a GetJobStates call")
	}

	req := legacymodels.StateRequest{
		ClientID: apiClient.ClientID,
		JobID:    jobID,
	}

	var res legacymodels.StateResponse
	var outerErr error

	for i := 0; i < APIRetryCount; i++ {
		shortTimeoutCtx, cancelFn := context.WithTimeout(ctx, time.Second*APIShortTimeoutSeconds)
		defer cancelFn()
		err := apiClient.DoPost(shortTimeoutCtx, "/api/v1/requester/states", req, &res)
		if err == nil {
			return res.State, nil
		} else {
			log.Ctx(ctx).Debug().Err(err).Msg("apiclient read state error")
			outerErr = err
		}
	}
	return model.JobState{}, outerErr
}

func (apiClient *APIClient) GetJobStateResolver() *legacy_job.StateResolver {
	jobLoader := func(ctx context.Context, jobID string) (model.Job, error) {
		j, _, err := apiClient.Get(ctx, jobID)
		if err != nil {
			return model.Job{}, fmt.Errorf("failed to load job %s: %w", jobID, err)
		}
		return j.Job, err
	}
	stateLoader := func(ctx context.Context, jobID string) (model.JobState, error) {
		return apiClient.GetJobState(ctx, jobID)
	}
	return legacy_job.NewStateResolver(jobLoader, stateLoader)
}

func (apiClient *APIClient) GetEvents(
	ctx context.Context,
	jobID string,
	options legacymodels.EventFilterOptions) (events []model.JobHistory, err error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester/publicapi.APIClient.GetEvents")
	defer span.End()

	if jobID == "" {
		return nil, fmt.Errorf("jobID must be non-empty in a GetEvents call")
	}

	req := legacymodels.EventsRequest{
		ClientID: apiClient.ClientID,
		JobID:    jobID,
		Options:  options,
	}

	// Test if the context has been canceled before making the request.
	var res legacymodels.EventsResponse
	var outerErr error

	for i := 0; i < APIRetryCount; i++ {
		shortTimeoutCtx, cancelFn := context.WithTimeout(ctx, time.Second*APIShortTimeoutSeconds)
		defer cancelFn()
		err = apiClient.DoPost(shortTimeoutCtx, "/api/v1/requester/events", req, &res)
		if err == nil {
			return res.Events, nil
		} else {
			log.Ctx(ctx).Debug().Err(err).Msg("apiclient read events error")
			outerErr = err
			if strings.Contains(err.Error(), "context canceled") {
				outerErr = bacerrors.NewContextCanceledError(ctx.Err().Error())
			}
		}
	}
	return nil, outerErr
}

func (apiClient *APIClient) GetResults(ctx context.Context, jobID string) (results []model.PublishedResult, err error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester/publicapi.APIClient.GetResults")
	defer span.End()

	if jobID == "" {
		return nil, fmt.Errorf("jobID must be non-empty in a GetResults call")
	}

	req := legacymodels.ResultsRequest{
		ClientID: apiClient.ClientID,
		JobID:    jobID,
	}

	var res legacymodels.ResultsResponse
	if err := apiClient.DoPost(ctx, "/api/v1/requester/results", req, &res); err != nil {
		return nil, err
	}

	return res.Results, nil
}

// Submit submits a new job to the node's transport.
func (apiClient *APIClient) Submit(
	ctx context.Context,
	j *model.Job,
) (*model.Job, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester/publicapi.APIClient.Submit")
	defer span.End()

	data := model.JobCreatePayload{
		ClientID:   apiClient.ClientID,
		APIVersion: j.APIVersion,
		Spec:       &j.Spec,
	}

	var res legacymodels.SubmitResponse
	err := apiClient.doPostSigned(ctx, "/api/v1/requester/submit", data, &res)
	if err != nil {
		return &model.Job{}, err
	}

	return res.Job, nil
}

// Logs will retrieve the address of an endpoint where a client connection can be
// made to stream the results of an execution back to a TTY
func (apiClient *APIClient) Logs(
	ctx context.Context,
	jobID string,
	executionID string,
	withHistory bool,
	follow bool) (*websocket.Conn, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester/publicapi.APIClient.Logs")
	defer span.End()

	if jobID == "" {
		return nil, fmt.Errorf("jobID must be non-empty in a logs call")
	}

	if executionID == "" {
		return nil, fmt.Errorf("executionID must be non-empty in a logs call")
	}

	// Check the existence of a job with the provided ID, whether it is a short or long ID.
	jobInfo, found, err := apiClient.Get(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, bacerrors.NewJobNotFound(jobID)
	}

	// We potentially used the short jobID which `Get` supports and so let's switch
	// to use the longer version.
	jobID = jobInfo.State.JobID

	// Create a payload before signing it with our local key (for verification on the
	// server).
	payload := model.LogsPayload{
		ClientID:    apiClient.ClientID,
		JobID:       jobID,
		ExecutionID: executionID,
		Tail:        withHistory,
		Follow:      follow,
	}

	req, err := signatures.SignRequest(apiClient.Signer, payload)
	if err != nil {
		return nil, err
	}

	u, _ := url.Parse(apiClient.BaseURI.String())
	u.Scheme = "ws"
	u.Path = "/api/v1/requester/logs"

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil) //nolint:bodyclose
	if err != nil {
		log.Ctx(ctx).Error().Msgf("Failed to dial to : %s", u.String())
		return nil, err
	}

	err = c.WriteJSON(req)
	if err != nil {
		log.Ctx(ctx).Error().Msg("Failed to write the JSON for the start of the logs request")
		return nil, err
	}

	return c, nil
}

func (apiClient *APIClient) Debug(ctx context.Context) (map[string]model.DebugInfo, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester/publicapi.APIClient.Debug")
	defer span.End()

	req := struct{}{}
	var res map[string]model.DebugInfo
	if err := apiClient.DoPost(ctx, "/api/v1/requester/debug", req, &res); err != nil {
		return res, err
	}

	return res, nil
}

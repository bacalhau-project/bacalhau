package publicapi

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// APIRetryCount - for some queries (like read events and read state)
// we want to fail early (10 seconds should be ample time)
// but retry a number of times - this is to avoid network
// flakes failing the canary
const APIRetryCount = 5
const APIShortTimeoutSeconds = 10

// RequesterAPIClient is a utility for interacting with a node's API server.
type RequesterAPIClient struct {
	publicapi.APIClient
}

// NewRequesterAPIClient returns a new client for a node's API server.
func NewRequesterAPIClient(host string, port uint16, path ...string) *RequesterAPIClient {
	return NewRequesterAPIClientFromClient(publicapi.NewAPIClient(host, port, path...))
}

// NewRequesterAPIClientFromClient returns a new client for a node's API server.
func NewRequesterAPIClientFromClient(baseClient *publicapi.APIClient) *RequesterAPIClient {
	return &RequesterAPIClient{
		APIClient: *baseClient,
	}
}

// List returns the list of jobs in the node's transport.
func (apiClient *RequesterAPIClient) List(
	ctx context.Context,
	idFilter string,
	includeTags []model.IncludedTag,
	excludeTags []model.ExcludedTag,
	maxJobs int,
	returnAll bool,
	sortBy string,
	sortReverse bool,
) (
	[]*model.JobWithInfo, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester/publicapi.RequesterAPIClient.List")
	defer span.End()

	req := listRequest{
		ClientID:    system.GetClientID(),
		MaxJobs:     maxJobs,
		JobID:       idFilter,
		IncludeTags: includeTags,
		ExcludeTags: excludeTags,
		ReturnAll:   returnAll,
		SortBy:      sortBy,
		SortReverse: sortReverse,
	}

	var res listResponse
	if err := apiClient.Post(ctx, APIPrefix+"list", req, &res); err != nil {
		e := err
		return nil, e
	}

	return res.Jobs, nil
}

// Cancel will request that the job with the specified ID is stopped. The JobInfo will be returned if the cancel
// was submitted. If no match is found, Cancel returns false with a nil error.
func (apiClient *RequesterAPIClient) Cancel(ctx context.Context, jobID string, reason string) (*model.JobState, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester/publicapi.RequesterAPIClient.Cancel")
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
		ClientID: system.GetClientID(),
		JobID:    jobID,
		Reason:   reason,
	}

	var res cancelResponse
	if err := apiClient.PostSigned(ctx, APIPrefix+"cancel", req, &res); err != nil {
		return &model.JobState{}, err
	}

	return res.State, nil
}

// Get returns job data for a particular job ID. If no match is found, Get returns false with a nil error.
func (apiClient *RequesterAPIClient) Get(ctx context.Context, jobID string) (*model.JobWithInfo, bool, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester/publicapi.RequesterAPIClient.Get")
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

func (apiClient *RequesterAPIClient) GetJobState(ctx context.Context, jobID string) (model.JobState, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester/publicapi.RequesterAPIClient.GetJobState")
	defer span.End()

	if jobID == "" {
		return model.JobState{}, fmt.Errorf("jobID must be non-empty in a GetJobStates call")
	}

	req := stateRequest{
		ClientID: system.GetClientID(),
		JobID:    jobID,
	}

	var res stateResponse
	var outerErr error

	for i := 0; i < APIRetryCount; i++ {
		shortTimeoutCtx, cancelFn := context.WithTimeout(ctx, time.Second*APIShortTimeoutSeconds)
		defer cancelFn()
		err := apiClient.Post(shortTimeoutCtx, APIPrefix+"states", req, &res)
		if err == nil {
			return res.State, nil
		} else {
			log.Ctx(ctx).Debug().Err(err).Msg("apiclient read state error")
			outerErr = err
		}
	}
	return model.JobState{}, outerErr
}

func (apiClient *RequesterAPIClient) GetJobStateResolver() *job.StateResolver {
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
	return job.NewStateResolver(jobLoader, stateLoader)
}

func (apiClient *RequesterAPIClient) GetEvents(
	ctx context.Context,
	jobID string,
	options EventFilterOptions) (events []model.JobHistory, err error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester/publicapi.RequesterAPIClient.GetEvents")
	defer span.End()

	if jobID == "" {
		return nil, fmt.Errorf("jobID must be non-empty in a GetEvents call")
	}

	req := eventsRequest{
		ClientID: system.GetClientID(),
		JobID:    jobID,
		Options:  options,
	}

	// Test if the context has been canceled before making the request.
	var res eventsResponse
	var outerErr error

	for i := 0; i < APIRetryCount; i++ {
		shortTimeoutCtx, cancelFn := context.WithTimeout(ctx, time.Second*APIShortTimeoutSeconds)
		defer cancelFn()
		err = apiClient.Post(shortTimeoutCtx, APIPrefix+"events", req, &res)
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

func (apiClient *RequesterAPIClient) GetResults(ctx context.Context, jobID string) (results []model.PublishedResult, err error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester/publicapi.RequesterAPIClient.GetResults")
	defer span.End()

	if jobID == "" {
		return nil, fmt.Errorf("jobID must be non-empty in a GetResults call")
	}

	req := resultsRequest{
		ClientID: system.GetClientID(),
		JobID:    jobID,
	}

	var res resultsResponse
	if err := apiClient.Post(ctx, APIPrefix+"results", req, &res); err != nil {
		return nil, err
	}

	return res.Results, nil
}

// Submit submits a new job to the node's transport.
func (apiClient *RequesterAPIClient) Submit(
	ctx context.Context,
	j *model.Job,
) (*model.Job, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester/publicapi.RequesterAPIClient.Submit")
	defer span.End()

	data := model.JobCreatePayload{
		ClientID:   system.GetClientID(),
		APIVersion: j.APIVersion,
		Spec:       &j.Spec,
	}

	var res submitResponse
	err := apiClient.PostSigned(ctx, APIPrefix+"submit", data, &res)
	if err != nil {
		return &model.Job{}, err
	}

	return res.Job, nil
}

func (apiClient *RequesterAPIClient) Approve(
	ctx context.Context,
	jobID string,
	response bidstrategy.BidStrategyResponse,
) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester/publicapi.RequesterAPIClient.Approve")
	defer span.End()

	data := bidstrategy.ModerateJobRequest{
		ClientID: system.GetClientID(),
		JobID:    jobID,
		Response: response,
	}

	return apiClient.PostSigned(ctx, APIPrefix+ApprovalRoute, data, nil)
}

// Logs will retrieve the address of an endpoint where a client connection can be
// made to stream the results of an execution back to a TTY
func (apiClient *RequesterAPIClient) Logs(
	ctx context.Context,
	jobID string,
	executionID string,
	withHistory bool,
	follow bool) (*websocket.Conn, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester/publicapi.RequesterAPIClient.Logs")
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
		ClientID:    system.GetClientID(),
		JobID:       jobID,
		ExecutionID: executionID,
		WithHistory: withHistory,
		Follow:      follow,
	}

	req, err := publicapi.SignRequest(payload)
	if err != nil {
		return nil, err
	}

	u, _ := url.Parse(apiClient.APIClient.BaseURI.String())
	u.Scheme = "ws"
	u.Path = APIPrefix + "logs"

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

func (apiClient *RequesterAPIClient) Debug(ctx context.Context) (map[string]model.DebugInfo, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester/publicapi.RequesterAPIClient.Debug")
	defer span.End()

	req := struct{}{}
	var res map[string]model.DebugInfo
	if err := apiClient.Post(ctx, APIPrefix+"debug", req, &res); err != nil {
		return res, err
	}

	return res, nil
}

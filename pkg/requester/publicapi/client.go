package publicapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
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
func NewRequesterAPIClient(baseURI string) *RequesterAPIClient {
	return NewRequesterAPIClientFromClient(publicapi.NewAPIClient(baseURI))
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
	[]*model.Job, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publicapi.List")
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

// Get returns job data for a particular job ID. If no match is found, Get returns false with a nil error.
func (apiClient *RequesterAPIClient) Get(ctx context.Context, jobID string) (*model.Job, bool, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publicapi.Get")
	defer span.End()

	if jobID == "" {
		return &model.Job{}, false, fmt.Errorf("jobID must be non-empty in a Get call")
	}

	jobsList, err := apiClient.List(ctx, jobID, model.IncludeAny, model.ExcludeNone, 1, false, "created_at", true)
	if err != nil {
		return &model.Job{}, false, err
	}

	if len(jobsList) > 0 {
		return jobsList[0], true, nil
	} else {
		return &model.Job{}, false, bacerrors.NewJobNotFound(jobID)
	}
}

func (apiClient *RequesterAPIClient) GetJobState(ctx context.Context, jobID string) (model.JobState, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publicapi.GetJobState")
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
			log.Debug().Err(err).Msg("apiclient read state error")
			outerErr = err
		}
	}
	return model.JobState{}, outerErr
}

func (apiClient *RequesterAPIClient) GetJobStateResolver() *job.StateResolver {
	jobLoader := func(ctx context.Context, jobID string) (*model.Job, error) {
		j, _, err := apiClient.Get(ctx, jobID)
		if err != nil {
			return &model.Job{}, fmt.Errorf("failed to load job %s: %w", jobID, err)
		}
		return j, err
	}
	stateLoader := func(ctx context.Context, jobID string) (model.JobState, error) {
		return apiClient.GetJobState(ctx, jobID)
	}
	return job.NewStateResolver(jobLoader, stateLoader)
}

func (apiClient *RequesterAPIClient) GetEvents(ctx context.Context, jobID string) (events []model.JobEvent, err error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publicapi.GetEvents")
	defer span.End()

	if jobID == "" {
		return nil, fmt.Errorf("jobID must be non-empty in a GetEvents call")
	}

	req := eventsRequest{
		ClientID: system.GetClientID(),
		JobID:    jobID,
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
			log.Debug().Err(err).Msg("apiclient read events error")
			outerErr = err
			if strings.Contains(err.Error(), "context canceled") {
				outerErr = bacerrors.NewContextCanceledError(ctx.Err().Error())
			}
		}
	}
	return nil, outerErr
}

func (apiClient *RequesterAPIClient) GetLocalEvents(ctx context.Context, jobID string) (localEvents []model.JobLocalEvent, err error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publicapi.GetLocalEvents")
	defer span.End()

	if jobID == "" {
		return nil, fmt.Errorf("jobID must be non-empty in a GetLocalEvents call")
	}

	req := localEventsRequest{
		ClientID: system.GetClientID(),
		JobID:    jobID,
	}

	var res localEventsResponse
	if err := apiClient.Post(ctx, APIPrefix+"local_events", req, &res); err != nil {
		return nil, err
	}

	return res.LocalEvents, nil
}

func (apiClient *RequesterAPIClient) GetResults(ctx context.Context, jobID string) (results []model.PublishedResult, err error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publicapi.GetResults")
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
	ctx, span := system.GetTracer().Start(ctx, "pkg/publicapi.Submit")
	defer span.End()

	data := model.JobCreatePayload{
		ClientID:   system.GetClientID(),
		APIVersion: j.APIVersion,
		Spec:       &j.Spec,
	}

	jsonData, err := model.JSONMarshalWithMax(data)
	if err != nil {
		return &model.Job{}, err
	}
	jsonRaw := json.RawMessage(jsonData)
	log.Ctx(ctx).Debug().RawJSON("json", jsonRaw).Msgf("jsonRaw")

	// sign the raw bytes representation of model.JobCreatePayload
	signature, err := system.SignForClient(jsonRaw)
	if err != nil {
		return &model.Job{}, err
	}
	log.Ctx(ctx).Debug().Str("signature", signature).Msgf("signature")

	var res submitResponse
	req := submitRequest{
		JobCreatePayload: &jsonRaw,
		ClientSignature:  signature,
		ClientPublicKey:  system.GetClientPublicKey(),
	}

	err = apiClient.Post(ctx, APIPrefix+"submit", req, &res)
	if err != nil {
		return &model.Job{}, err
	}

	return res.Job, nil
}

func (apiClient *RequesterAPIClient) Debug(ctx context.Context) (map[string]model.DebugInfo, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publicapi.Debug")
	defer span.End()

	req := struct{}{}
	var res map[string]model.DebugInfo
	if err := apiClient.Post(ctx, APIPrefix+"debug", req, &res); err != nil {
		return res, err
	}

	return res, nil
}

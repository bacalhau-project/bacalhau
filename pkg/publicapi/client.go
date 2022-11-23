package publicapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/closer"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// APIClient is a utility for interacting with a node's API server.
type APIClient struct {
	BaseURI string

	client *http.Client
}

// NewAPIClient returns a new client for a node's API server.
func NewAPIClient(baseURI string) *APIClient {
	return &APIClient{
		BaseURI: baseURI,

		client: &http.Client{
			Timeout: 300 * time.Second,
			Transport: otelhttp.NewTransport(nil,
				otelhttp.WithSpanOptions(
					trace.WithAttributes(
						attribute.String("clientID", system.GetClientID()),
					),
				),
			),
		},
	}
}

// Alive calls the node's API server health check.
func (apiClient *APIClient) Alive(ctx context.Context) (bool, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publicapi.Alive")
	defer span.End()

	var body io.Reader
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiClient.BaseURI+"/livez", body)
	if err != nil {
		return false, nil
	}
	res, err := apiClient.client.Do(req) //nolint:bodyclose // golangcilint is dumb - this is closed
	if err != nil {
		return false, nil
	}
	defer closer.DrainAndCloseWithLogOnError(ctx, "apiClient response", res.Body)

	return res.StatusCode == http.StatusOK, nil
}

// List returns the list of jobs in the node's transport.
func (apiClient *APIClient) List(ctx context.Context, idFilter string, maxJobs int, returnAll bool, sortBy string, sortReverse bool) (
	[]*model.Job, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publicapi.List")
	defer span.End()

	req := listRequest{
		ClientID:    system.GetClientID(),
		MaxJobs:     maxJobs,
		JobID:       idFilter,
		ReturnAll:   returnAll,
		SortBy:      sortBy,
		SortReverse: sortReverse,
	}

	var res listResponse
	if err := apiClient.post(ctx, "list", req, &res); err != nil {
		e := err
		return nil, e
	}

	return res.Jobs, nil
}

// Get returns job data for a particular job ID. If no match is found, Get returns false with a nil error.
func (apiClient *APIClient) Get(ctx context.Context, jobID string) (*model.Job, bool, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publicapi.Get")
	defer span.End()

	if jobID == "" {
		return &model.Job{}, false, fmt.Errorf("jobID must be non-empty in a Get call")
	}

	jobsList, err := apiClient.List(ctx, jobID, 1, false, "created_at", true)
	if err != nil {
		return &model.Job{}, false, err
	}

	if len(jobsList) > 0 {
		return jobsList[0], true, nil
	} else {
		return &model.Job{}, false, bacerrors.NewJobNotFound(jobID)
	}
}

func (apiClient *APIClient) GetJobState(ctx context.Context, jobID string) (model.JobState, error) {
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
	if err := apiClient.post(ctx, "states", req, &res); err != nil {
		return model.JobState{}, err
	}

	return res.State, nil
}

func (apiClient *APIClient) GetJobStateResolver() *job.StateResolver {
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

func (apiClient *APIClient) GetEvents(ctx context.Context, jobID string) (events []model.JobEvent, err error) {
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
	err = apiClient.post(ctx, "events", req, &res)
	if err != nil {
		if strings.Contains(err.Error(), "context canceled") {
			return nil, bacerrors.NewContextCanceledError(ctx.Err().Error())
		}

		log.Debug().Err(err).Msg("request error")
		return nil, err
	}

	return res.Events, nil
}

func (apiClient *APIClient) GetLocalEvents(ctx context.Context, jobID string) (localEvents []model.JobLocalEvent, err error) {
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
	if err := apiClient.post(ctx, "local_events", req, &res); err != nil {
		return nil, err
	}

	return res.LocalEvents, nil
}

func (apiClient *APIClient) GetResults(ctx context.Context, jobID string) (results []model.PublishedResult, err error) {
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
	if err := apiClient.post(ctx, "results", req, &res); err != nil {
		return nil, err
	}

	return res.Results, nil
}

// Submit submits a new job to the node's transport.
func (apiClient *APIClient) Submit(
	ctx context.Context,
	j *model.Job,
	buildContext *bytes.Buffer,
) (*model.Job, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publicapi.Submit")
	defer span.End()

	data := model.JobCreatePayload{
		ClientID: system.GetClientID(),
		Job:      j,
	}

	if buildContext != nil {
		data.Context = base64.StdEncoding.EncodeToString(buildContext.Bytes())
	}

	jsonData, err := model.JSONMarshalWithMax(data)
	if err != nil {
		return &model.Job{}, err
	}

	signature, err := system.SignForClient(jsonData)
	if err != nil {
		return &model.Job{}, err
	}

	var res submitResponse
	req := submitRequest{
		Data:            data,
		ClientSignature: signature,
		ClientPublicKey: system.GetClientPublicKey(),
	}

	err = apiClient.post(ctx, "submit", req, &res)
	if err != nil {
		return &model.Job{}, err
	}

	return res.Job, nil
}

// Submit submits a new job to the node's transport.
func (apiClient *APIClient) Version(ctx context.Context) (*model.BuildVersionInfo, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publicapi.Version")
	defer span.End()

	req := listRequest{
		ClientID: system.GetClientID(),
	}

	var res versionResponse
	if err := apiClient.post(ctx, "version", req, &res); err != nil {
		return nil, err
	}

	return res.VersionInfo, nil
}

func (apiClient *APIClient) post(ctx context.Context, api string, reqData, resData interface{}) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publicapi.post")
	defer span.End()

	var body bytes.Buffer
	var err error
	if err = json.NewEncoder(&body).Encode(reqData); err != nil {
		return bacerrors.NewResponseUnknownError(fmt.Errorf("publicapi: error encoding request body: %v", err))
	}

	addr := fmt.Sprintf("%s/%s", apiClient.BaseURI, api)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, addr, &body)
	if err != nil {
		return bacerrors.NewResponseUnknownError(fmt.Errorf("publicapi: error creating post request: %v", err))
	}
	req.Header.Set("Content-type", "application/json")
	req.Close = true // don't keep connections lying around

	var res *http.Response
	res, err = apiClient.client.Do(req)
	if err != nil {
		errString := err.Error()
		if errorResponse, ok := err.(*bacerrors.ErrorResponse); ok {
			return errorResponse
		} else if errString == "context canceled" {
			return bacerrors.NewContextCanceledError(err.Error())
		} else {
			return bacerrors.NewResponseUnknownError(fmt.Errorf("publicapi: after posting request: %v", err))
		}
	}

	defer func() {
		if err = res.Body.Close(); err != nil {
			err = fmt.Errorf("error closing response body: %v", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		var responseBody []byte
		responseBody, err = io.ReadAll(res.Body)
		if err != nil {
			return bacerrors.NewResponseUnknownError(fmt.Errorf("publicapi: error reading response body: %v", err))
		}

		var serverError *bacerrors.ErrorResponse
		if err = model.JSONUnmarshalWithMax(responseBody, &serverError); err != nil {
			return bacerrors.NewResponseUnknownError(fmt.Errorf("publicapi: after posting request: %v",
				string(responseBody)))
		}

		if !reflect.DeepEqual(serverError, bacerrors.BacalhauErrorInterface(nil)) {
			return serverError
		}
	}

	err = json.NewDecoder(res.Body).Decode(resData)
	if err != nil {
		if err == io.EOF {
			return nil // No error, just no data
		} else {
			return bacerrors.NewResponseUnknownError(fmt.Errorf("publicapi: error decoding response body: %v", err))
		}
	}

	return nil
}

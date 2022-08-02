package publicapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
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
func (apiClient *APIClient) Alive() (bool, error) {
	var body io.Reader
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, apiClient.BaseURI+"/livez", body)
	if err != nil {
		return false, nil
	}
	res, err := apiClient.client.Do(req)
	if err != nil {
		return false, nil
	}
	defer res.Body.Close()

	return res.StatusCode == http.StatusOK, nil
}

// List returns the list of jobs in the node's transport.
func (apiClient *APIClient) List(ctx context.Context) (map[string]executor.Job, error) {
	req := listRequest{
		ClientID: system.GetClientID(),
	}

	var res listResponse
	if err := apiClient.post(ctx, "list", req, &res); err != nil {
		return nil, err
	}

	return res.Jobs, nil
}

// Get returns job data for a particular job ID. If no match is found, Get returns false with a nil error.
// TODO(optimisation): implement with separate API call, don't filter list
func (apiClient *APIClient) Get(ctx context.Context, jobID string) (job executor.Job, foundJob bool, err error) {
	if jobID == "" {
		return executor.Job{}, false, fmt.Errorf("jobID must be non-empty in a Get call")
	}

	jobs, err := apiClient.List(ctx)
	if err != nil {
		return executor.Job{}, false, err
	}

	// TODO: make this deterministic, return the first match alphabetically
	for _, job = range jobs { //nolint:gocritic
		if strings.HasPrefix(job.ID, jobID) {
			return job, true, nil
		}
	}

	return executor.Job{}, false, nil
}

func (apiClient *APIClient) GetJobState(ctx context.Context, jobID string) (states executor.JobState, err error) {
	if jobID == "" {
		return executor.JobState{}, fmt.Errorf("jobID must be non-empty in a GetJobStates call")
	}

	req := stateRequest{
		ClientID: system.GetClientID(),
		JobID:    jobID,
	}

	var res stateResponse
	if err := apiClient.post(ctx, "states", req, &res); err != nil {
		return executor.JobState{}, err
	}

	return res.State, nil
}

func (apiClient *APIClient) GetJobStateResolver() *job.StateResolver {
	jobLoader := func(ctx context.Context, jobID string) (executor.Job, error) {
		job, ok, err := apiClient.Get(ctx, jobID)
		if !ok {
			return executor.Job{}, fmt.Errorf("no job found with id %s", jobID)
		}
		return job, err
	}
	stateLoader := func(ctx context.Context, jobID string) (executor.JobState, error) {
		return apiClient.GetJobState(ctx, jobID)
	}
	return job.NewStateResolver(jobLoader, stateLoader)
}

func (apiClient *APIClient) GetEvents(ctx context.Context, jobID string) (events []executor.JobEvent, err error) {
	if jobID == "" {
		return nil, fmt.Errorf("jobID must be non-empty in a GetEvents call")
	}

	req := eventsRequest{
		ClientID: system.GetClientID(),
		JobID:    jobID,
	}

	var res eventsResponse
	if err := apiClient.post(ctx, "events", req, &res); err != nil {
		return nil, err
	}

	return res.Events, nil
}

func (apiClient *APIClient) GetLocalEvents(ctx context.Context, jobID string) (localEvents []executor.JobLocalEvent, err error) {
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

func (apiClient *APIClient) GetResults(ctx context.Context, jobID string) (results []storage.StorageSpec, err error) {
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
	spec executor.JobSpec,
	deal executor.JobDeal,
	buildContext *bytes.Buffer,
) (executor.Job, error) {
	data := executor.JobCreatePayload{
		ClientID: system.GetClientID(),
		Spec:     spec,
		Deal:     deal,
	}

	if buildContext != nil {
		data.Context = base64.StdEncoding.EncodeToString(buildContext.Bytes())
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return executor.Job{}, err
	}

	signature, err := system.SignForClient(jsonData)
	if err != nil {
		return executor.Job{}, err
	}

	var res submitResponse
	req := submitRequest{
		Data:            data,
		ClientSignature: signature,
		ClientPublicKey: system.GetClientPublicKey(),
	}

	if err := apiClient.post(ctx, "submit", req, &res); err != nil {
		return executor.Job{}, err
	}

	return res.Job, nil
}

// Submit submits a new job to the node's transport.
func (apiClient *APIClient) Version(ctx context.Context) (*executor.VersionInfo, error) {
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
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(reqData); err != nil {
		return fmt.Errorf("publicapi: error encoding request body: %v", err)
	}

	addr := fmt.Sprintf("%s/%s", apiClient.BaseURI, api)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, addr, &body)
	if err != nil {
		return fmt.Errorf("publicapi: error creating post request: %v", err)
	}
	req.Header.Set("Content-type", "application/json")
	req.Close = true // don't keep connections lying around

	res, err := apiClient.client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if err != nil {
		return fmt.Errorf("publicapi: error sending post request: %v", err)
	}

	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Error().Msgf("error closing response body: %v", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		body, err := io.ReadAll(res.Body)
		if err == nil { // not critical if this fails
			log.Error().Msgf(
				"publicapi: %d body returned from API server: %s", res.StatusCode, string(body))
		}

		return fmt.Errorf(
			"publicapi: received non-200 status: %d", res.StatusCode)
	}

	if err := json.NewDecoder(res.Body).Decode(resData); err != nil {
		return fmt.Errorf("publicapi: error decoding response body: %v", err)
	}

	return nil
}

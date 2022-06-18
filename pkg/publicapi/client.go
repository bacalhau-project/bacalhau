package publicapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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
			Timeout:   300 * time.Second,
			Transport: otelhttp.NewTransport(nil),
		},
	}
}

// Alive calls the node's API server health check.
func (apiClient *APIClient) Alive() (bool, error) {
	res, err := http.Get(apiClient.BaseURI + "/livez")
	if err != nil {
		return false, nil
	}

	return res.StatusCode == http.StatusOK, nil
}

// List returns the list of jobs in the node's transport.
func (apiClient *APIClient) List(ctx context.Context) (
	map[string]*types.Job, error) {

	var req listRequest
	var res listResponse

	if err := apiClient.post(ctx, "list", req, &res); err != nil {
		return nil, err
	}

	return res.Jobs, nil
}

// Get returns job data for a particular job ID.
// TODO(optimisation): implement with separate API call, don't filter list
func (apiClient *APIClient) Get(ctx context.Context, jobID string) (
	*types.Job, bool, error) {

	jobs, err := apiClient.List(ctx)
	if err != nil {
		return nil, false, err
	}

	for _, job := range jobs {
		// TODO: could have multiple matches in jobs, right? is this bad?
		if strings.HasPrefix(job.Id, jobID) {
			return job, true, nil
		}
	}

	return nil, false, nil
}

// Submit submits a new job to the node's transport.
func (apiClient *APIClient) Submit(ctx context.Context, spec *types.JobSpec,
	deal *types.JobDeal) (*types.Job, error) {

	var res submitResponse
	req := submitRequest{
		Spec: spec,
		Deal: deal,
	}

	if err := apiClient.post(ctx, "submit", req, &res); err != nil {
		return nil, err
	}

	return res.Job, nil
}

func (apiClient *APIClient) post(ctx context.Context, api string,
	reqData interface{}, resData interface{}) error {

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(reqData); err != nil {
		return fmt.Errorf("publicapi: error encoding request body: %v", err)
	}

	addr := fmt.Sprintf("%s/%s", apiClient.BaseURI, api)
	req, err := http.NewRequestWithContext(ctx, "POST", addr, &body)
	if err != nil {
		return fmt.Errorf("publicapi: error creating post request: %v", err)
	}
	req.Header.Set("Content-type", "application/json")

	res, err := apiClient.client.Do(req)
	if err != nil {
		return fmt.Errorf("publicapi: error sending post request: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(res.Body)
		if err == nil { // not critical if this fails
			log.Error().Msgf(
				"publicapi: non-200 body returned from API server: %s", string(body))
		}

		return fmt.Errorf(
			"publicapi: received non-200 status: %d", res.StatusCode)
	}

	if err := json.NewDecoder(res.Body).Decode(resData); err != nil {
		return fmt.Errorf("publicapi: error decoding response body: %v", err)
	}

	return nil
}

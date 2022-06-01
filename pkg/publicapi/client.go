package publicapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/types"
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
		},
	}
}

// Healthy calls the node's API server health check.
func (apiClient *APIClient) Healthy() (bool, error) {
	res, err := http.Get(apiClient.BaseURI + "/health")
	if err != nil {
		return false, nil
	}

	return res.StatusCode == http.StatusOK, nil
}

// List returns the list of jobs in the node's transport.
func (apiClient *APIClient) List() (map[string]*types.Job, error) {
	var req listRequest
	var res listResponse

	if err := apiClient.post("list", req, &res); err != nil {
		return nil, err
	}

	return res.Jobs, nil
}

// Get returns job data for a particular job ID.
// TODO(optimisation): implement with separate API call, don't filter list
func (apiClient *APIClient) Get(jobID string) (*types.Job, error) {
	jobs, err := apiClient.List()
	if err != nil {
		return nil, err
	}

	for _, job := range jobs {
		// TODO: could have multiple matches in jobs, right? is this bad?
		if strings.HasPrefix(job.Id, jobID) {
			return job, nil
		}
	}

	return nil, fmt.Errorf("could not find job with ID: %s", jobID)
}

// Submit submits a new job to the node's transport.
func (apiClient *APIClient) Submit(
	spec *types.JobSpec,
	deal *types.JobDeal,
) (*types.Job, error) {
	var res submitResponse
	req := submitRequest{
		Spec: spec,
		Deal: deal,
	}

	if err := apiClient.post("submit", req, &res); err != nil {
		return nil, err
	}

	return res.Job, nil
}

func (apiClient *APIClient) post(
	api string,
	reqData interface{},
	resData interface{},
) error {
	addr := fmt.Sprintf("%s/%s", apiClient.BaseURI, api)

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(reqData); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", addr, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-type", "application/json")

	res, err := apiClient.client.Do(req)
	if err != nil {
		return err
	}

	return json.NewDecoder(res.Body).Decode(resData)
}

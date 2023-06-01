package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"path/filepath"

	"go.uber.org/multierr"
	"golang.org/x/mod/sumdb/dirhash"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
	"github.com/bacalhau-project/bacalhau/pkg/verifier/results"
)

// ExternalVerifier provides verification proposals to some service outside of
// the core Bacalhau system, and allows those services to give a verification
// result.
type ExternalVerifier struct {
	provider publisher.PublisherProvider
	results  *results.Results
	webhook  *url.URL
}

func NewExternalVerifier(
	provider publisher.PublisherProvider,
	webhook *url.URL,
) (verifier.Verifier, error) {
	results, err := results.NewResults()
	return &ExternalVerifier{
		provider: provider,
		results:  results,
		webhook:  webhook,
	}, err
}

type ExternalVerificationRequest struct {
	ExecutionID model.ExecutionID `json:"executionId"`
	Results     []spec.Storage    `json:"results"`
	Callback    *url.URL          `json:"callback"`
}

type ExternalVerificationResponse struct {
	ClientID      string                    `json:"clientId"`
	Verifications []verifier.VerifierResult `json:"verifications"`
}

func (e ExternalVerificationResponse) GetClientID() string {
	return e.ClientID
}

// IsInstalled implements verifier.Verifier
func (v *ExternalVerifier) IsInstalled(context.Context) (bool, error) {
	return true, nil
}

// GetProposal implements verifier.Verifier
func (v *ExternalVerifier) GetProposal(ctx context.Context, job model.Job, executionID string, resultPath string) ([]byte, error) {
	_, span := system.NewSpan(ctx, system.GetTracer(), "pkg/verifier.ExternalVerifier.GetProposal")
	defer span.End()

	store, err := v.provider.Get(ctx, job.Spec.PublisherSpec.Type)
	if err != nil {
		return nil, err
	}

	hash, err := dirhash.HashDir(resultPath, "results", dirhash.Hash1)
	if err != nil {
		return nil, err
	}

	spec, err := store.PublishResult(ctx, executionID, job, resultPath)
	if err != nil {
		return nil, err
	}
	// TODO metadata is required on (all?) some storage specs
	spec.Metadata.Put("hash", hash)

	var size, count int64
	err = filepath.WalkDir(resultPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		count += 1
		size += info.Size()
		return nil
	})
	if err != nil {
		return nil, err
	}
	// TODO metadata is required on (all?) some storage specs
	spec.Metadata.Put("count", fmt.Sprint(count))
	spec.Metadata.Put("size", fmt.Sprint(size))

	return json.Marshal(&spec)
}

// GetResultPath implements verifier.Verifier
func (v *ExternalVerifier) GetResultPath(ctx context.Context, executionID string, job model.Job) (string, error) {
	_, span := system.NewSpan(ctx, system.GetTracer(), "pkg/verifier.ExternalVerifier.GetResultPath")
	defer span.End()

	return v.results.EnsureResultsDir(executionID)
}

// Verify implements verifier.Verifier
func (v *ExternalVerifier) Verify(
	ctx context.Context,
	request verifier.VerifierRequest,
) (results []verifier.VerifierResult, err error) {
	_, span := system.NewSpan(ctx, system.GetTracer(), "pkg/verifier.ExternalVerifier.Verify")
	defer span.End()

	err = verifier.ValidateExecutions(request)
	if err != nil {
		return nil, err
	}

	specs := make([]spec.Storage, len(request.Executions))
	for i, state := range request.Executions {
		// TODO will need to use the concrete spec unmarshaller probably with a switch statement on spec schema type
		// alternativly we could change the type of VerificationProposal to be more descriptive seems like its basically
		// a storage spec... maybe make it that?
		err = multierr.Append(err, json.Unmarshal(state.VerificationProposal, &specs[i]))
	}
	if err != nil {
		return nil, err
	}

	requestData, err := json.Marshal(&request)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, v.webhook.String(), bytes.NewReader(requestData))
	if err != nil {
		return nil, err
	}

	//nolint:bodyclose // Closed in DrainAndCloseWithLogOnError
	response, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer closer.DrainAndCloseWithLogOnError(ctx, v.webhook.String(), response.Body)

	if response.StatusCode == http.StatusAccepted {
		// Webhook has accepted our request and will respond later.
		return nil, nil
	} else if response.StatusCode == http.StatusOK {
		// Webhook has responded immediately.
		err = json.NewDecoder(response.Body).Decode(&results)
		return results, err
	} else {
		return nil, fmt.Errorf("bad HTTP response: %d %s", response.StatusCode, response.Status)
	}
}

var _ verifier.Verifier = (*ExternalVerifier)(nil)

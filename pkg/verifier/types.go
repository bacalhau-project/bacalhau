package verifier

import (
	"context"
	"net/url"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type EncrypterFunction func(ctx context.Context, data []byte, publicKeyBytes []byte) ([]byte, error)
type DecrypterFunction func(ctx context.Context, data []byte) ([]byte, error)

type VerifierRequest struct {
	JobID      string
	Deal       model.Deal
	Executions []model.ExecutionState
	Callback   *url.URL
}

type VerifierResult struct {
	ExecutionID model.ExecutionID
	Verified    bool
}

// Returns a verifier that can be used to verify a job.
type VerifierProvider = model.Provider[model.Verifier, Verifier]

// Verifier is an interface representing something that can verify the results
// of a job.
type Verifier interface {
	model.Providable

	// compute node
	//
	// return the local file path where the output of a local execution
	// should live - this is called by the executor to prepare
	// output volumes when running a job and the publisher when uploading
	// the results after verification
	GetResultPath(
		ctx context.Context,
		executionID string,
		job model.Job,
	) (string, error)

	// compute node
	//
	// the executor has completed the job and produced a local folder of results
	// the verifier will now "process" this local folder into the result
	// that will be broadcast back to the network
	// For example - the "resultsHash" verifier will hash the folder
	// and encrypt that hash with the public key of the requester
	GetProposal(
		ctx context.Context,
		job model.Job,
		executionID string,
		resultPath string,
	) ([]byte, error)

	// requester node
	//
	// once we've decided that a job is complete - we verify the results reported
	// by the compute nodes - what this actually does is up to the verifier but
	// it's highly likely that a verifier implementation has a controller attached
	// and so can trigger state transitions (such as results accepted / rejected)
	//
	// IsJobComplete must return true otherwise this function will error
	Verify(
		ctx context.Context,
		request VerifierRequest,
	) ([]VerifierResult, error)
}

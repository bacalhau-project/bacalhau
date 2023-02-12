package verifier

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

type EncrypterFunction func(ctx context.Context, data []byte, publicKeyBytes []byte) ([]byte, error)
type DecrypterFunction func(ctx context.Context, data []byte) ([]byte, error)

type VerifierResult struct {
	Execution model.ExecutionState
	Verified  bool
}

// Returns a verifier that can be used to verify a job.
type VerifierProvider interface {
	model.Provider[model.Verifier, Verifier]
}

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
	GetShardResultPath(
		ctx context.Context,
		shard model.JobShard,
	) (string, error)

	// compute node
	//
	// the executor has completed the job and produced a local folder of results
	// the verifier will now "process" this local folder into the result
	// that will be broadcast back to the network
	// For example - the "resultsHash" verifier will hash the folder
	// and encrypt that hash with the public key of the requester
	GetShardProposal(
		ctx context.Context,
		shard model.JobShard,
		shardResultPath string,
	) ([]byte, error)

	// requester node
	//
	// once we've decided that a job is complete - we verify the results reported
	// by the compute nodes - what this actually does is up to the verifier but
	// it's highly likely that a verifier implementation has a controller attached
	// and so can trigger state transitions (such as results accepted / rejected)
	// for each of the shards reported
	//
	// IsJobComplete must return true otherwise this function will error
	VerifyShard(
		ctx context.Context,
		shard model.JobShard,
		executions []model.ExecutionState,
	) ([]VerifierResult, error)
}

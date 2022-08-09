package verifier

import (
	"context"
)

type VerifierResult struct {
	HostID     string
	JobID      string
	ShardIndex int
	Verified   bool
	Error      error
}

// Verifier is an interface representing something that can verify the results
// of a job.
type Verifier interface {
	// tells you if the required software is installed on this machine
	IsInstalled(context.Context) (bool, error)

	// compute node
	//
	// return the local file path where the output of a local execution
	// should live - this is called by the executor to prepare
	// output volumes when running a job and the publisher when uploading
	// the results after verification
	GetShardResultPath(
		ctx context.Context,
		jobID string,
		shardIndex int,
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
		jobID string,
		shardIndex int,
		shardResultPath string,
	) (string, error)

	// requester node
	//
	// do we think that enough executions have occured to call this job "complete"
	// there should be at least 1 result per shard but it's really up to the verifier
	// to decide that a job has "completed"
	IsJobComplete(
		ctx context.Context,
		jobID string,
	) (bool, error)

	// requester node
	//
	// once we've decided that a job is complete - we verify the results reported
	// by the compute nodes - what this actually does is up to the verifier but
	// it's highly likely that a verifier implementation has a controller attached
	// and so can trigger state transitions (such as results accepted / rejected)
	// for each of the shards reported
	//
	// IsJobComplete must return true otherwise this function will error
	VerifyJob(
		ctx context.Context,
		jobID string,
	) ([]VerifierResult, error)
}

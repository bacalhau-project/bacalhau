package noop

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/filecoin-project/bacalhau/pkg/verifier/results"
)

type NoopVerifier struct {
	results *results.Results
}

func NewNoopVerifier(
	_ context.Context, cm *system.CleanupManager,
) (*NoopVerifier, error) {
	results, err := results.NewResults()
	if err != nil {
		return nil, err
	}

	cm.RegisterCallback(func() error {
		if err := results.Close(); err != nil {
			return fmt.Errorf("unable to remove results folder: %w", err)
		}
		return nil
	})
	return &NoopVerifier{
		results: results,
	}, nil
}

func (noopVerifier *NoopVerifier) IsInstalled(context.Context) (bool, error) {
	return true, nil
}

func (noopVerifier *NoopVerifier) GetShardResultPath(
	_ context.Context,
	shard model.JobShard,
) (string, error) {
	return noopVerifier.results.EnsureShardResultsDir(shard.Job.Metadata.ID, shard.Index)
}

func (noopVerifier *NoopVerifier) GetShardProposal(
	context.Context,
	model.JobShard,
	string,
) ([]byte, error) {
	return []byte{}, nil
}

func (noopVerifier *NoopVerifier) VerifyShard(
	ctx context.Context,
	shard model.JobShard,
	executionStates []model.ExecutionState,
) ([]verifier.VerifierResult, error) {
	_, span := system.NewSpan(ctx, system.GetTracer(), "pkg/verifier.NoopVerifier.VerifyShard")
	defer span.End()

	err := verifier.ValidateExecutions(shard, executionStates)
	if err != nil {
		return nil, err
	}

	var verifierResults []verifier.VerifierResult
	for _, execution := range executionStates { //nolint:gocritic
		verifierResults = append(verifierResults, verifier.VerifierResult{
			Execution: execution,
			Verified:  true,
		})
	}
	return verifierResults, nil
}

// Compile-time check that NoopVerifier implements the correct interface:
var _ verifier.Verifier = (*NoopVerifier)(nil)

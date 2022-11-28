package noop

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/filecoin-project/bacalhau/pkg/verifier/results"
)

// Verifier provider that always return NoopVerifier regardless of requested verifier type
type NoopVerifierProvider struct {
	noopVerifier *NoopVerifier
}

func NewNoopVerifierProvider(noopVerifier *NoopVerifier) *NoopVerifierProvider {
	return &NoopVerifierProvider{
		noopVerifier: noopVerifier,
	}
}

func (p *NoopVerifierProvider) GetVerifier(ctx context.Context, jobVerifier model.Verifier) (verifier.Verifier, error) {
	if jobVerifier != model.VerifierNoop {
		return nil, fmt.Errorf("no verifier available for type %s. Only VerifierNoop is supported", jobVerifier)
	}
	return p.noopVerifier, nil
}

// Check if a verifier is available or not
func (p *NoopVerifierProvider) HasVerifier(ctx context.Context, verifierType model.Verifier) bool {
	_, err := p.GetVerifier(ctx, verifierType)
	return err == nil
}

type NoopVerifier struct {
	stateResolver *job.StateResolver
	results       *results.Results
}

func NewNoopVerifier(
	_ context.Context, cm *system.CleanupManager,
	resolver *job.StateResolver,
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
		stateResolver: resolver,
		results:       results,
	}, nil
}

func (noopVerifier *NoopVerifier) IsInstalled(context.Context) (bool, error) {
	return true, nil
}

func (noopVerifier *NoopVerifier) GetShardResultPath(
	_ context.Context,
	shard model.JobShard,
) (string, error) {
	return noopVerifier.results.EnsureShardResultsDir(shard.Job.ID, shard.Index)
}

func (noopVerifier *NoopVerifier) GetShardProposal(
	context.Context,
	model.JobShard,
	string,
) ([]byte, error) {
	return []byte{}, nil
}

// each shard must have >= concurrency states
// and they must be either JobStateError or JobStateVerifying
func (noopVerifier *NoopVerifier) IsExecutionComplete(
	ctx context.Context,
	shard model.JobShard,
) (bool, error) {
	return noopVerifier.stateResolver.CheckShardStates(ctx, shard, func(
		shardStates []model.JobShardState,
		concurrency int,
	) (bool, error) {
		return noopVerifier.results.CheckShardStates(shardStates, concurrency)
	})
}

func (noopVerifier *NoopVerifier) VerifyShard(
	ctx context.Context,
	shard model.JobShard,
) ([]verifier.VerifierResult, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/verifier/noop/NoopVerifier.VerifyShard")
	defer span.End()

	results := []verifier.VerifierResult{}
	jobState, err := noopVerifier.stateResolver.GetJobState(ctx, shard.Job.ID)
	if err != nil {
		return results, err
	}
	shardStates := job.GetStatesForShardIndex(jobState, shard.Index)
	if len(shardStates) == 0 {
		return nil, fmt.Errorf("job (%s) has no shard state for shard index %d", shard.Job.ID, shard.Index)
	}

	for _, shardState := range shardStates { //nolint:gocritic
		if shardState.State != model.JobStateVerifying {
			continue
		}
		results = append(results, verifier.VerifierResult{
			JobID:      shard.Job.ID,
			NodeID:     shardState.NodeID,
			ShardIndex: shardState.ShardIndex,
			Verified:   true,
		})
	}
	return results, nil
}

// Compile-time check that NoopVerifier implements the correct interface:
var _ verifier.VerifierProvider = (*NoopVerifierProvider)(nil)
var _ verifier.Verifier = (*NoopVerifier)(nil)

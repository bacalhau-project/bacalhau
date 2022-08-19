package noop

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/storage/util"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

type NoopVerifier struct {
	StateResolver *job.StateResolver
	// where do we copy the results from jobs temporarily?
	ResultsDir string
}

func NewNoopVerifier(
	cm *system.CleanupManager,
	resolver *job.StateResolver,
) (*NoopVerifier, error) {
	dir, err := ioutil.TempDir("", "bacalhau-noop-verifier")
	if err != nil {
		return nil, err
	}

	return &NoopVerifier{
		StateResolver: resolver,
		ResultsDir:    dir,
	}, nil
}

func (noopVerifier *NoopVerifier) IsInstalled(ctx context.Context) (bool, error) {
	return true, nil
}

func (noopVerifier *NoopVerifier) GetShardResultPath(
	ctx context.Context,
	jobID string,
	shardIndex int,
) (string, error) {
	return noopVerifier.ensureShardResultsDir(jobID, shardIndex)
}

func (noopVerifier *NoopVerifier) GetShardProposal(
	ctx context.Context,
	jobID string,
	shardIndex int,
	shardResultPath string,
) ([]byte, error) {
	return []byte{}, nil
}

// each shard must have >= concurrency states
// and they must be either JobStateError or JobStateVerifying
func (noopVerifier *NoopVerifier) IsExecutionComplete(
	ctx context.Context,
	jobID string,
) (bool, error) {
	return noopVerifier.StateResolver.CheckShardStates(ctx, jobID, func(
		shardStates []executor.JobShardState,
		concurrency int,
	) (bool, error) {
		if len(shardStates) < concurrency {
			return false, nil
		}
		for _, state := range shardStates {
			if state.State != executor.JobStateError && state.State != executor.JobStateVerifying {
				return false, nil
			}
		}
		return true, nil
	})
}

func (noopVerifier *NoopVerifier) VerifyJob(
	ctx context.Context,
	jobID string,
) ([]verifier.VerifierResult, error) {
	results := []verifier.VerifierResult{}
	jobState, err := noopVerifier.StateResolver.GetJobState(ctx, jobID)
	if err != nil {
		return results, err
	}
	for _, shardState := range job.FlattenShardStates(jobState) {
		if shardState.State != executor.JobStateVerifying {
			continue
		}
		results = append(results, verifier.VerifierResult{
			JobID:      jobID,
			NodeID:     shardState.NodeID,
			ShardIndex: shardState.ShardIndex,
			Verified:   true,
		})
	}
	return results, nil
}

func (noopVerifier *NoopVerifier) getShardResultsDir(jobID string, shardIndex int) string {
	return fmt.Sprintf("%s/%s/%d", noopVerifier.ResultsDir, jobID, shardIndex)
}

func (noopVerifier *NoopVerifier) ensureShardResultsDir(jobID string, shardIndex int) (string, error) {
	dir := noopVerifier.getShardResultsDir(jobID, shardIndex)
	err := os.MkdirAll(dir, util.OS_ALL_RWX)
	info, _ := os.Stat(dir)
	log.Trace().Msgf("Created job results dir (%s). Permissions: %s", dir, info.Mode())
	return dir, err
}

func newSpan(ctx context.Context, apiName string) (context.Context, trace.Span) {
	return system.Span(ctx, "verifier/noop", apiName)
}

// Compile-time check that NoopVerifier implements the correct interface:
var _ verifier.Verifier = (*NoopVerifier)(nil)

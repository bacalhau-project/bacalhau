package noop

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/storage/util"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

type NoopVerifier struct {
	JobLoader   job.JobLoader
	StateLoader job.StateLoader
	// where do we copy the results from jobs temporarily?
	ResultsDir string
}

func NewNoopVerifier(
	cm *system.CleanupManager,
	jobLoader job.JobLoader,
	stateLoader job.StateLoader,
) (*NoopVerifier, error) {
	dir, err := ioutil.TempDir("", "bacalhau-noop-verifier")
	if err != nil {
		return nil, err
	}

	return &NoopVerifier{
		JobLoader:   jobLoader,
		StateLoader: stateLoader,
		ResultsDir:  dir,
	}, nil
}

func (v *NoopVerifier) IsInstalled(ctx context.Context) (bool, error) {
	return true, nil
}

func (v *NoopVerifier) GetShardResultPath(
	ctx context.Context,
	jobID string,
	shardIndex int,
) (string, error) {
	return v.ensureShardResultsDir(jobID, shardIndex)
}

func (v *NoopVerifier) GetProposal(
	ctx context.Context,
	jobID string,
	shardIndex int,
	shardResultPath string,
) ([]byte, error) {
	return []byte{}, nil
}

func (v *NoopVerifier) IsJobComplete(
	ctx context.Context,
	jobID string,
) (bool, error) {
	return false, nil
}

func (v *NoopVerifier) VerifyJob(
	ctx context.Context,
	jobID string,
) ([]verifier.VerifierResult, error) {
	return []verifier.VerifierResult{}, nil
}

func (v *NoopVerifier) getShardResultsDir(jobID string, shardIndex int) string {
	return fmt.Sprintf("%s/%s/%d", v.ResultsDir, jobID, shardIndex)
}

func (v *NoopVerifier) ensureShardResultsDir(jobID string, shardIndex int) (string, error) {
	dir := v.getShardResultsDir(jobID, shardIndex)
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

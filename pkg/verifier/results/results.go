package results

import (
	"fmt"
	"os"

	"github.com/filecoin-project/bacalhau/pkg/storage/util"
	"github.com/rs/zerolog/log"
)

type Results struct {
	// where do we copy the results from jobs temporarily?
	ResultsDir string
}

func NewResults() (*Results, error) {
	dir, err := os.MkdirTemp("", "bacalhau-results")
	if err != nil {
		return nil, err
	}
	return &Results{
		ResultsDir: dir,
	}, nil
}

func (results *Results) GetShardResultsDir(jobID string, shardIndex int) string {
	return fmt.Sprintf("%s/%s/%d", results.ResultsDir, jobID, shardIndex)
}

func (results *Results) EnsureShardResultsDir(jobID string, shardIndex int) (string, error) {
	dir := results.GetShardResultsDir(jobID, shardIndex)
	err := os.MkdirAll(dir, util.OS_ALL_RWX)
	if err != nil {
		return "", fmt.Errorf("error creating results dir %s: %w", dir, err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("error getting results dir %s info: %w", dir, err)
	}
	log.Trace().Msgf("Created job results dir (%s). Permissions: %s", dir, info.Mode())
	return dir, err
}

func (results *Results) Close() error {
	return os.RemoveAll(results.ResultsDir)
}

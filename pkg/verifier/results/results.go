package results

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage/util"
	"github.com/rs/zerolog/log"
)

type Results struct {
	// where do we copy the results from jobs temporarily?
	ResultsDir string
}

func NewResults() (*Results, error) {
	dir, err := ioutil.TempDir("", "bacalhau-results")
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
	info, _ := os.Stat(dir)
	log.Trace().Msgf("Created job results dir (%s). Permissions: %s", dir, info.Mode())
	return dir, err
}

func (results *Results) CheckShardStates(
	shardStates []model.JobShardState,
	concurrency int,
) (bool, error) {
	if len(shardStates) < concurrency {
		return false, nil
	}
	hasExecutedCount := 0
	for _, state := range shardStates { //nolint:gocritic
		if state.State == model.JobStateError || state.State == model.JobStateVerifying {
			hasExecutedCount++
		}
	}
	return hasExecutedCount >= concurrency, nil
}

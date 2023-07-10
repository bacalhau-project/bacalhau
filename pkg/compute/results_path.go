package compute

import (
	"fmt"
	"os"

	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/rs/zerolog/log"
)

type ResultsPath struct {
	// where do we copy the results from jobs temporarily?
	ResultsDir string
}

func NewResultsPath() (*ResultsPath, error) {
	dir, err := os.MkdirTemp("", "bacalhau-results")
	if err != nil {
		return nil, err
	}
	return &ResultsPath{
		ResultsDir: dir,
	}, nil
}

func (results *ResultsPath) getResultsDir(executionID string) string {
	return fmt.Sprintf("%s/%s", results.ResultsDir, executionID)
}

// PrepareResultsDir creates a temporary directory to store the results of a job execution.
func (results *ResultsPath) PrepareResultsDir(executionID string) (string, error) {
	dir := results.getResultsDir(executionID)
	err := os.MkdirAll(dir, util.OS_ALL_RWX)
	if err != nil {
		return "", fmt.Errorf("error creating results dir %s: %w", dir, err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("error getting results dir %s info: %w", dir, err)
	}
	log.Trace().Msgf("Created execution results dir (%s). Permissions: %s", dir, info.Mode())
	return dir, err
}

// EnsureResultsDir ensures that the results directory exists.
func (results *ResultsPath) EnsureResultsDir(executionID string) (string, error) {
	dir := results.getResultsDir(executionID)
	_, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("error getting results dir %s info: %w", dir, err)
	}
	return dir, err
}

func (results *ResultsPath) Close() error {
	if _, err := os.Stat(results.ResultsDir); os.IsNotExist(err) {
		return nil
	}

	return os.RemoveAll(results.ResultsDir)
}

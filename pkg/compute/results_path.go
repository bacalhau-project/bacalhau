package compute

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/rs/zerolog/log"
)

const (
	OutputDir  = "output"
	LogsDir    = "logs"
	ResultsDir = "results"
)

func ExecutionLogsDir(resultsRootDir string, executionID string) string {
	return filepath.Join(resultsRootDir, LogsDir)
}

func ExecutionResultsDir(resultsRootDir string, executionID string) string {
	return filepath.Join(resultsRootDir, ResultsDir)
}

// Execution results folder structure
//
//	→ rootDir
//		→ OutputDir								<- root for the results of all executions
//			→ $execution_id						<- execution output directory
//				→ LogsDir
//				→ ResultsDir
type ResultsPath struct {
	OutputDir string
}

func NewResultsPath(rootDir string) (*ResultsPath, error) {
	outputDir := filepath.Join(rootDir, OutputDir)
	if err := prepareDir(outputDir); err != nil {
		return nil, err
	}

	return &ResultsPath{
		OutputDir: outputDir,
	}, nil
}

func (r *ResultsPath) ExecutionOutputDir(executionID string) string {
	return filepath.Join(r.OutputDir, executionID)
}

func (r *ResultsPath) PrepareExecutionOutputDir(executionID string) (string, error) {
	// execution results root directory
	executionResultsRootPath := r.ExecutionOutputDir(executionID)
	if err := prepareDir(executionResultsRootPath); err != nil {
		return "", err
	}

	// execution logs directory
	logsPath := ExecutionLogsDir(executionResultsRootPath, executionID)
	if err := prepareDir(logsPath); err != nil {
		return "", err
	}

	// execution results directory
	resultsPath := ExecutionResultsDir(executionResultsRootPath, executionID)
	if err := prepareDir(resultsPath); err != nil {
		return "", err
	}

	return executionResultsRootPath, nil
}

func (r *ResultsPath) Close() error {
	log.Debug().Str("path", r.OutputDir).Msg("removing root results dir")
	return os.RemoveAll(r.OutputDir)
}

// Creates a folder at given path with rwx------ permissions.
// Parent directory must exist.
func prepareDir(path string) error {
	log.Debug().Str("path", path).Msg("creating results dir")
	err := os.Mkdir(path, util.OS_USER_RWX) // Results directories should only be accessible by the Bacalhau user
	if err != nil {
		return fmt.Errorf("error creating results dir %s: %w", path, err)
	}
	return nil
}

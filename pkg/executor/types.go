package executor

import (
	"context"
	"encoding/json"
	"io"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

// Returns a executor for the given engine type
type ExecutorProvider = model.Provider[model.Engine, Executor]

// Executor represents an execution provider, which can execute jobs on some
// kind of backend, such as a docker daemon.
type Executor interface {
	model.Providable

	// A BidStrategy that should return a positive response if the executor
	// could run the job or a negative response otherwise.
	GetSemanticBidStrategy(context.Context) (bidstrategy.SemanticBidStrategy, error)

	GetResourceBidStrategy(ctx context.Context) (bidstrategy.ResourceBidStrategy, error)

	// GetOutputStream retrieves a muxed stream from the executor
	GetOutputStream(ctx context.Context, executionID string, withHistory bool, follow bool) (io.ReadCloser, error)

	// run the given job - it's expected that we have already prepared the job
	// this will return a local filesystem path to the jobs results
	Run(ctx context.Context, Args *RunCommandRequest) (*model.RunCommandResult, error)
}

type RunCommandRequest struct {
	JobID                string
	ExecutionID          string
	Resources            model.ResourceUsageConfig
	Network              model.NetworkConfig
	Outputs              []model.StorageSpec
	Inputs               []storage.PreparedStorage
	EnvironmentVariables map[string]string
	ResultsDir           string
	EngineParams         *Arguments
}

// ExecutorParams is a stub for pluggable engines
type Arguments struct {
	Params []byte
}

func EncodeArguments(in interface{}) (*Arguments, error) {
	b, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	return &Arguments{Params: b}, nil
}

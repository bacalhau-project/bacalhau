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
	// A Providable is something that a Provider can check for installation status
	model.Providable

	bidstrategy.SemanticBidStrategy
	bidstrategy.ResourceBidStrategy

	GetOutputStream(ctx context.Context, executionID string, withHistory bool, follow bool) (io.ReadCloser, error)

	// TODO refactor to add non-blocking methods for start, stopping, and waiting on executions:
	// Details in: https://github.com/bacalhau-project/bacalhau/issues/2702

	Run(ctx context.Context, Args *RunCommandRequest) (*model.RunCommandResult, error)
	Cancel(ctx context.Context, id string) error
}

type RunCommandRequest struct {
	JobID        string
	ExecutionID  string
	Resources    model.ResourceUsageConfig
	Network      model.NetworkConfig
	Outputs      []model.StorageSpec
	Inputs       []storage.PreparedStorage
	ResultsDir   string
	EngineParams *Arguments
	OutputLimits OutputLimits
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

package executor

import (
	"context"
	"io"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

// Returns a executor for the given engine type
type ExecutorProvider = provider.Provider[Executor]

// Executor represents an execution provider, which can execute jobs on some
// kind of backend, such as a docker daemon.
type Executor interface {
	// A Providable is something that a Provider can check for installation status
	provider.Providable

	bidstrategy.SemanticBidStrategy
	bidstrategy.ResourceBidStrategy

	GetOutputStream(ctx context.Context, executionID string, withHistory bool, follow bool) (io.ReadCloser, error)

	// TODO refactor to add non-blocking methods for start, stopping, and waiting on executions:
	// Details in: https://github.com/bacalhau-project/bacalhau/issues/2702

	Run(ctx context.Context, Args *RunCommandRequest) (*models.RunCommandResult, error)
	Cancel(ctx context.Context, id string) error
}

type RunCommandRequest struct {
	JobID        string
	ExecutionID  string
	Resources    *models.Resources
	Network      *models.NetworkConfig
	Outputs      []*models.ResultPath
	Inputs       []storage.PreparedStorage
	ResultsDir   string
	EngineParams *models.SpecConfig
	OutputLimits OutputLimits
}

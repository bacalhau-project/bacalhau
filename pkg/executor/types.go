package executor

import (
	"context"
	"fmt"
	"io"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

// ExecutorProvider returns a executor for the given engine type
type ExecutorProvider = provider.Provider[Executor]

// Executor serves as an execution manager for running jobs on a specific backend, such as a Docker daemon.
// It provides a comprehensive set of methods to initiate, monitor, terminate, and retrieve output streams for executions.
type Executor interface {
	// A Providable is something that a Provider can check for installation status
	provider.Providable

	bidstrategy.SemanticBidStrategy
	bidstrategy.ResourceBidStrategy

	// Start initiates an execution for the given RunCommandRequest.
	// It returns an error if the execution already exists and is in a started or terminal state.
	// Implementations may also return other errors based on resource limitations or internal faults.
	Start(ctx context.Context, request *RunCommandRequest) error

	// Run initiates and waits for the completion of an execution for the given RunCommandRequest.
	// It returns a RunCommandResult and an error if any part of the operation fails.
	// Specifically, it will return an error if the execution already exists and is in a started or terminal state.
	Run(ctx context.Context, args *RunCommandRequest) (*models.RunCommandResult, error)

	// Wait waits for the completion of an execution identified by its executionID.
	// It returns a channel that emits the result once the execution is complete.
	// Returns an error if the execution does not exist or is already in a terminal state.
	Wait(ctx context.Context, executionID string) (<-chan *models.RunCommandResult, error)

	// Cancel attempts to cancel an ongoing execution identified by its executionID.
	// Returns an error if the execution does not exist or is already in a terminal state.
	Cancel(ctx context.Context, executionID string) error

	// GetOutputStream provides a stream of output for an ongoing or completed execution identified by its executionID.
	// The 'withHistory' flag indicates whether to include historical data in the stream.
	// The 'follow' flag indicates whether the stream should continue to send data as it is produced.
	// Returns an io.ReadCloser to read the output stream and an error if the operation fails.
	// Specifically, it will return an error if the execution does not exist.
	GetOutputStream(ctx context.Context, executionID string, withHistory bool, follow bool) (io.ReadCloser, error)
}

// RunCommandRequest encapsulates the parameters required to initiate a job execution.
// It includes identifiers, resource requirements, network configurations, and various other settings.
type RunCommandRequest struct {
	JobID        string                    // Unique identifier for the job.
	ExecutionID  string                    // Unique identifier for a specific execution of the job.
	Resources    *models.Resources         // Resource requirements like CPU, Memory, GPU, Disk.
	Network      *models.NetworkConfig     // Network configuration for the execution.
	Outputs      []*models.ResultPath      // Paths where the execution should store its outputs.
	Inputs       []storage.PreparedStorage // Prepared storage elements that are used as inputs.
	ResultsDir   string                    // Directory where results should be stored.
	EngineParams *models.SpecConfig        // Engine-specific configuration parameters.
	OutputLimits OutputLimits              // Output size limits for the execution.
}

// Error variables for execution states.
var (
	// AlreadyStartedErr is returned when trying to start an already started execution.
	AlreadyStartedErr = fmt.Errorf("execution already started")

	// AlreadyCompleteErr is returned when action is attempted on an execution that is already complete.
	AlreadyCompleteErr = fmt.Errorf("execution already complete")

	// NotFoundErr is returned when the execution ID provided does not match any existing execution.
	NotFoundErr = fmt.Errorf("execution not found")
)

package executor

import (
	"context"
	"io"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

const ExecComponentName = "Executor"

// ExecProvider returns a executor for the given engine type
type ExecProvider = provider.Provider[Executor]

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

	// Wait monitors the completion of an execution identified by its executionID.
	// It returns two channels:
	// 1. A channel that emits the execution result once the task is complete.
	// 2. An error channel that relays any issues encountered, such as when the
	//    execution is non-existent or has already concluded.
	Wait(ctx context.Context, executionID string) (<-chan *models.RunCommandResult, <-chan error)

	// Cancel attempts to cancel an ongoing execution identified by its executionID.
	// Returns an error if the execution does not exist or is already in a terminal state.
	Cancel(ctx context.Context, executionID string) error

	// GetLogStream provides a stream of output for an ongoing or completed execution identified by its executionID.
	// The 'withHistory' flag indicates whether to include historical data in the stream.
	// The 'follow' flag indicates whether the stream should continue to send data as it is produced.
	// Returns an io.ReadCloser to read the output stream and an error if the operation fails.
	// Specifically, it will return an error if the execution does not exist.
	GetLogStream(ctx context.Context, request LogStreamRequest) (io.ReadCloser, error)
}

// LogStreamRequest encapsulates the parameters required to retrieve a log stream.
type LogStreamRequest struct {
	JobID       string
	ExecutionID string
	Tail        bool
	Follow      bool
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

// Common Error Codes for Executor
const (
	ExecutionAlreadyStarted   bacerrors.ErrorCode = "ExecutionAlreadyStarted"
	ExecutionAlreadyCancelled bacerrors.ErrorCode = "ExecutionAlreadyCancelled"
	ExecutionAlreadyComplete  bacerrors.ErrorCode = "ExecutionAlreadyComplete"
	ExecutionNotFound         bacerrors.ErrorCode = "ExecutionNotFound"
	ExecutorSpecValidationErr bacerrors.ErrorCode = "ExecutorSpecValidationErr"
)

func NewExecutorError(code bacerrors.ErrorCode, message string) bacerrors.Error {
	return bacerrors.New("%s", message).WithCode(code).WithComponent(ExecComponentName)
}

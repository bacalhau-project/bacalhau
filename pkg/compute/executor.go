package compute

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
)

type BaseExecutorParams struct {
	ID                     string
	Callback               Callback
	Store                  store.ExecutionStore
	Storages               storage.StorageProvider
	Executors              executor.ExecutorProvider
	ResultsPath            ResultsPath
	Publishers             publisher.PublisherProvider
	FailureInjectionConfig model.FailureInjectionComputeConfig
}

// BaseExecutor is the base implementation for backend service.
// All operations are executed asynchronously, and a callback is used to notify the caller of the result.
type BaseExecutor struct {
	ID               string
	callback         Callback
	store            store.ExecutionStore
	cancellers       generic.SyncMap[string, context.CancelFunc]
	Storages         storage.StorageProvider
	executors        executor.ExecutorProvider
	publishers       publisher.PublisherProvider
	resultsPath      ResultsPath
	failureInjection model.FailureInjectionComputeConfig
}

func NewBaseExecutor(params BaseExecutorParams) *BaseExecutor {
	return &BaseExecutor{
		ID:               params.ID,
		callback:         params.Callback,
		store:            params.Store,
		Storages:         params.Storages,
		executors:        params.Executors,
		publishers:       params.Publishers,
		failureInjection: params.FailureInjectionConfig,
		resultsPath:      params.ResultsPath,
	}
}

func PrepareRunArguments(
	ctx context.Context,
	strgprovider storage.StorageProvider,
	execution store.Execution,
	resultsDir string,
	cleanup *system.CleanupManager,
) (*executor.RunCommandRequest, error) {
	inputVolumes, err := storage.ParallelPrepareStorage(ctx, strgprovider, execution.Job.Spec.Inputs...)
	if err != nil {
		return nil, fmt.Errorf("failed to prepate storage for execution: %w", err)
	}

	cleanup.RegisterCallbackWithContext(func(ctx context.Context) error {
		if err := storage.ParallelCleanStorage(ctx, strgprovider, inputVolumes); err != nil {
			return fmt.Errorf("cleaning up job inputs: %w", err)
		}
		return nil
	})

	var engineArgs interface{}
	engineArgs = execution.Job.Spec.Docker
	if execution.Job.Spec.Engine == model.EngineWasm {
		importModuleVolumes, err := storage.ParallelPrepareStorage(ctx, strgprovider, execution.Job.Spec.Wasm.ImportModules...)
		if err != nil {
			return nil, err
		}
		cleanup.RegisterCallbackWithContext(func(ctx context.Context) error {
			if err := storage.ParallelCleanStorage(ctx, strgprovider, importModuleVolumes); err != nil {
				return fmt.Errorf("cleaning up wasm import modules: %w", err)
			}
			return nil
		})

		entryModuleVolumes, err := storage.ParallelPrepareStorage(ctx, strgprovider, execution.Job.Spec.Wasm.EntryModule)
		if err != nil {
			return nil, err
		}
		cleanup.RegisterCallbackWithContext(func(ctx context.Context) error {
			if err := storage.ParallelCleanStorage(ctx, strgprovider, entryModuleVolumes); err != nil {
				return fmt.Errorf("cleaning up wasm entry modules: %w", err)
			}
			return nil
		})
		engineArgs = &wasm.Arguments{
			EntryPoint:           execution.Job.Spec.Wasm.EntryPoint,
			Parameters:           execution.Job.Spec.Wasm.Parameters,
			EnvironmentVariables: execution.Job.Spec.Wasm.EnvironmentVariables,
			EntryModule:          entryModuleVolumes[0],
			ImportModules:        importModuleVolumes,
		}
	}
	args, err := executor.EncodeArguments(engineArgs)
	if err != nil {
		return nil, err
	}
	return &executor.RunCommandRequest{
		JobID:        execution.Job.ID(),
		ExecutionID:  execution.ID,
		Resources:    execution.Job.Spec.Resources,
		Network:      execution.Job.Spec.Network,
		Outputs:      execution.Job.Spec.Outputs,
		Inputs:       inputVolumes,
		ResultsDir:   resultsDir,
		EngineParams: args,
		OutputLimits: executor.OutputLimits{
			MaxStdoutFileLength:   system.MaxStdoutFileLength,
			MaxStdoutReturnLength: system.MaxStdoutReturnLength,
			MaxStderrFileLength:   system.MaxStderrFileLength,
			MaxStderrReturnLength: system.MaxStderrReturnLength,
		},
	}, nil
}

// Run the execution after it has been accepted, and propose a result to the requester to be verified.
func (e *BaseExecutor) Run(ctx context.Context, execution store.Execution) (err error) {
	ctx = log.Ctx(ctx).With().
		Str("job", execution.Job.ID()).
		Str("execution", execution.ID).
		Logger().WithContext(ctx)

	ctx, cancel := context.WithCancel(ctx)
	e.cancellers.Put(execution.ID, cancel)
	defer func() {
		if cancel, found := e.cancellers.Get(execution.ID); found {
			e.cancellers.Delete(execution.ID)
			cancel()
		}
	}()

	operation := "Running"
	defer func() {
		if err != nil {
			e.handleFailure(ctx, execution, err, operation)
		}
	}()

	log.Ctx(ctx).Debug().Msg("Running execution")
	if err := e.store.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:   execution.ID,
		ExpectedState: store.ExecutionStateBidAccepted,
		NewState:      store.ExecutionStateRunning,
	}); err != nil {
		return err
	}

	resultFolder, err := e.resultsPath.PrepareResultsDir(execution.ID)
	if err != nil {
		return fmt.Errorf("failed to get result path: %w", err)
	}

	jobExecutor, err := e.executors.Get(ctx, execution.Job.Spec.Engine)
	if err != nil {
		return fmt.Errorf("failed to get executor %s: %w", execution.Job.Spec.Engine, err)
	}

	if e.failureInjection.IsBadActor {
		return fmt.Errorf("i am a baaad node. i failed execution %s", execution.ID)
	}

	runCommandCleanup := system.NewCleanupManager()
	runCommandArguments, err := PrepareRunArguments(ctx, e.Storages, execution, resultFolder, runCommandCleanup)
	if err != nil {
		return err
	}
	defer runCommandCleanup.Cleanup(ctx)

	runCommandResult, err := jobExecutor.Run(ctx, runCommandArguments)
	if err != nil {
		jobsFailed.Add(ctx, 1)
		log.Ctx(ctx).Error().Err(err).Msg("failed to run execution")
		return err
	}
	jobsCompleted.Add(ctx, 1)

	if err := e.store.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:   execution.ID,
		ExpectedState: store.ExecutionStateRunning,
		NewState:      store.ExecutionStatePublishing,
	}); err != nil {
		return err
	}

	operation = "Publishing"
	return e.publish(ctx, execution, resultFolder, runCommandResult)
}

// Publish the result of an execution after it has been verified.
func (e *BaseExecutor) publish(ctx context.Context, execution store.Execution,
	resultFolder string, result *model.RunCommandResult) (err error) {
	log.Ctx(ctx).Debug().Msgf("Publishing execution %s", execution.ID)

	jobPublisher, err := e.publishers.Get(ctx, execution.Job.Spec.PublisherSpec.Type)
	if err != nil {
		err = fmt.Errorf("failed to get publisher %s: %w", execution.Job.Spec.PublisherSpec.Type, err)
		return
	}
	publishedResult, err := jobPublisher.PublishResult(ctx, execution.ID, execution.Job, resultFolder)
	if err != nil {
		err = fmt.Errorf("failed to publish result: %w", err)
		return
	}

	log.Ctx(ctx).Debug().
		Str("execution", execution.ID).
		Str("cid", publishedResult.CID).
		Msg("Execution published")

	err = e.store.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:   execution.ID,
		ExpectedState: store.ExecutionStatePublishing,
		NewState:      store.ExecutionStateCompleted,
	})
	if err != nil {
		return
	}

	log.Ctx(ctx).Debug().Msgf("Cleaning up result folder for %s: %s", execution.ID, resultFolder)
	err = os.RemoveAll(resultFolder)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to remove results folder at %s", resultFolder)
	}

	e.callback.OnRunComplete(ctx, RunResult{
		ExecutionMetadata: NewExecutionMetadata(execution),
		RoutingMetadata: RoutingMetadata{
			SourcePeerID: e.ID,
			TargetPeerID: execution.RequesterNodeID,
		},
		PublishResult:    publishedResult,
		RunCommandResult: result,
	})
	return err
}

// Cancel the execution.
func (e *BaseExecutor) Cancel(ctx context.Context, execution store.Execution) (err error) {
	defer func() {
		if err != nil {
			e.handleFailure(ctx, execution, err, "Canceling")
		}
	}()

	log.Ctx(ctx).Debug().Str("Execution", execution.ID).Msg("Canceling execution")
	if cancel, found := e.cancellers.Get(execution.ID); found {
		e.cancellers.Delete(execution.ID)
		cancel()
	}

	e.callback.OnCancelComplete(ctx, CancelResult{
		ExecutionMetadata: NewExecutionMetadata(execution),
		RoutingMetadata: RoutingMetadata{
			SourcePeerID: e.ID,
			TargetPeerID: execution.RequesterNodeID,
		},
	})
	return err
}

func (e *BaseExecutor) handleFailure(ctx context.Context, execution store.Execution, err error, operation string) {
	log.Ctx(ctx).Error().Err(err).Msgf("%s execution %s failed", operation, execution.ID)
	updateError := e.store.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID: execution.ID,
		NewState:    store.ExecutionStateFailed,
		Comment:     err.Error(),
	})

	if updateError != nil {
		log.Ctx(ctx).Error().Err(updateError).Msgf("Failed to update execution (%s) state to failed: %s", execution.ID, updateError)
	} else {
		e.callback.OnComputeFailure(ctx, ComputeError{
			ExecutionMetadata: NewExecutionMetadata(execution),
			RoutingMetadata: RoutingMetadata{
				SourcePeerID: e.ID,
				TargetPeerID: execution.RequesterNodeID,
			},
			Err: err.Error(),
		})
	}
}

// compile-time interface check
var _ Executor = (*BaseExecutor)(nil)

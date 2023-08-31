package compute

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/models"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	wasmmodels "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/models"
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
	execution *models.Execution,
	resultsDir string,
	cleanup *system.CleanupManager,
) (*executor.RunCommandRequest, error) {
	inputVolumes, err := storage.ParallelPrepareStorage(ctx, strgprovider, execution.Job.Task().InputSources...)
	if err != nil {
		return nil, fmt.Errorf("failed to prepate storage for execution: %w", err)
	}

	cleanup.RegisterCallbackWithContext(func(ctx context.Context) error {
		if err := storage.ParallelCleanStorage(ctx, strgprovider, inputVolumes); err != nil {
			return fmt.Errorf("cleaning up job inputs: %w", err)
		}
		return nil
	})

	var engineArgs *models.SpecConfig
	// TODO wasm requires special handling because its engine arguments are storage specs, and we need to
	// download them before passing it to the wasm executor
	/*
		The more general solution is make the WASM executor aware of which fields in the spec.Inputs StorageSpec
		are WASM modules. We would need to alter the WASM EngineSpec such that it can reference values from
		spec.Inputs with its EntryModule and ImportModules fields.
		(I suspect future implementations of an EngineSpec will need this ability - referencing specific
		inputs via their arguments - docker image comes to mind as a potential candidate).

		In #2675 we modified the Compute Node to initialize and download all spec.Inputs to local storage
		before passing it to the executor. Previously executors were responsible for downloading their inputs to
		local storage, and running the job. With our shift towards pluggable executors in #2637 configuring executor
		plugins to handle the download of different storage specs seems impractical
		(@wdbaruni's comment: https://github.com/bacalhau-project/bacalhau/pull/2637#issuecomment-1625739030
		provides more context on the need for the change).
	*/
	if execution.Job.Task().Engine.IsType(models.EngineWasm) {
		wasmEngine, err := wasmmodels.DecodeSpec(execution.Job.Task().Engine)
		if err != nil {
			return nil, err
		}
		importModuleVolumes, err := storage.ParallelPrepareStorage(ctx, strgprovider, wasmEngine.ImportModules...)
		if err != nil {
			return nil, err
		}
		cleanup.RegisterCallbackWithContext(func(ctx context.Context) error {
			if err := storage.ParallelCleanStorage(ctx, strgprovider, importModuleVolumes); err != nil {
				return fmt.Errorf("cleaning up wasm import modules: %w", err)
			}
			return nil
		})

		entryModuleVolumes, err := storage.ParallelPrepareStorage(ctx, strgprovider, wasmEngine.EntryModule)
		if err != nil {
			return nil, err
		}
		cleanup.RegisterCallbackWithContext(func(ctx context.Context) error {
			if err := storage.ParallelCleanStorage(ctx, strgprovider, entryModuleVolumes); err != nil {
				return fmt.Errorf("cleaning up wasm entry modules: %w", err)
			}
			return nil
		})
		engineArgs = &models.SpecConfig{
			Type:   models.EngineWasm,
			Params: wasmEngine.ToArguments(entryModuleVolumes[0], importModuleVolumes...).ToMap(),
		}
	} else {
		engineArgs = execution.Job.Task().Engine
	}
	return &executor.RunCommandRequest{
		JobID:        execution.Job.ID,
		ExecutionID:  execution.ID,
		Resources:    execution.TotalAllocatedResources(),
		Network:      execution.Job.Task().Network,
		Outputs:      execution.Job.Task().ResultPaths,
		Inputs:       inputVolumes,
		ResultsDir:   resultsDir,
		EngineParams: engineArgs,
		OutputLimits: executor.OutputLimits{
			MaxStdoutFileLength:   system.MaxStdoutFileLength,
			MaxStdoutReturnLength: system.MaxStdoutReturnLength,
			MaxStderrFileLength:   system.MaxStderrFileLength,
			MaxStderrReturnLength: system.MaxStderrReturnLength,
		},
	}, nil
}

func (e *BaseExecutor) Start(ctx context.Context, state store.LocalExecutionState) (err error) {
	execution := state.Execution
	ctx = log.Ctx(ctx).With().
		Str("job", execution.Job.ID).
		Str("execution", execution.ID).
		Logger().WithContext(ctx)

	operation := "Running"
	defer func() {
		if err != nil {
			e.handleFailure(ctx, state, err, operation)
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

	jobExecutor, err := e.executors.Get(ctx, execution.Job.Task().Engine.Type)
	if err != nil {
		return fmt.Errorf("failed to get executor %s: %w", execution.Job.Task().Engine, err)
	}

	if e.failureInjection.IsBadActor {
		return fmt.Errorf("i am a baaad node. i failed execution %s", execution.ID)
	}

	runCommandCleanup := system.NewCleanupManager()
	runCommandArguments, err := PrepareRunArguments(ctx, e.Storages, execution, resultFolder, runCommandCleanup)
	if err != nil {
		return err
	}
	// TODO this fucks everything up
	//defer runCommandCleanup.Cleanup(ctx)

	if err := jobExecutor.Start(ctx, runCommandArguments); err != nil {
		jobsFailed.Add(ctx, 1)
		log.Ctx(ctx).Error().Err(err).Msg("failed to start execution")
		return err
	}
	return nil
}

func (e *BaseExecutor) Wait(ctx context.Context, localExecutionState store.LocalExecutionState) (err error) {
	operation := "Publishing"
	defer func() {
		if err != nil {
			// TODO this needs to consider the case that an execution was canceled rather than failed.
			e.handleFailure(ctx, localExecutionState, err, operation)
		}
	}()
	execution := localExecutionState.Execution
	jobExecutor, err := e.executors.Get(ctx, execution.Job.Task().Engine.Type)
	if err != nil {
		return fmt.Errorf("failed to get executor %s: %w", execution.Job.Task().Engine, err)
	}

	waitCh, err := jobExecutor.Wait(ctx, execution.ID)
	if err != nil {
		jobsFailed.Add(ctx, 1)
		log.Ctx(ctx).Error().Err(err).Msg("failed to wait on execution")
		return err
	}
	var runCommandResult *models.RunCommandResult
	select {
	case <-ctx.Done():
		return ctx.Err()
	case res := <-waitCh:
		runCommandResult = res
	}
	if runCommandResult.ErrorMsg != "" {
		return fmt.Errorf("result returned from wait contains error: %s", runCommandResult.ErrorMsg)
	}

	jobsCompleted.Add(ctx, 1)

	if err := e.store.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:   execution.ID,
		ExpectedState: store.ExecutionStateRunning,
		NewState:      store.ExecutionStatePublishing,
	}); err != nil {
		return err
	}

	resultsDir, err := e.resultsPath.EnsureResultsDir(execution.ID)
	if err != nil {
		return err
	}
	return e.publish(ctx, localExecutionState, resultsDir, runCommandResult)

}

// Run the execution after it has been accepted, and propose a result to the requester to be verified.
func (e *BaseExecutor) Run(ctx context.Context, localExecutionState store.LocalExecutionState) (err error) {
	if err := e.Start(ctx, localExecutionState); err != nil {
		return err
	}
	if err := e.Wait(ctx, localExecutionState); err != nil {
		return err
	}
	return nil
}

// Publish the result of an execution after it has been verified.
func (e *BaseExecutor) publish(ctx context.Context, localExecutionState store.LocalExecutionState,
	resultFolder string, result *models.RunCommandResult) (err error) {
	execution := localExecutionState.Execution
	log.Ctx(ctx).Debug().Msgf("Publishing execution %s", execution.ID)

	jobPublisher, err := e.publishers.Get(ctx, execution.Job.Task().Publisher.Type)
	if err != nil {
		err = fmt.Errorf("failed to get publisher %s: %w", execution.Job.Task().Publisher.Type, err)
		return
	}
	publishedResult, err := jobPublisher.PublishResult(ctx, execution.ID, *execution.Job, resultFolder)
	if err != nil {
		err = fmt.Errorf("failed to publish result: %w", err)
		return
	}

	log.Ctx(ctx).Debug().
		Str("execution", execution.ID).
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
			TargetPeerID: localExecutionState.RequesterNodeID,
		},
		PublishResult:    &publishedResult,
		RunCommandResult: result,
	})
	return err
}

// Cancel the execution.
func (e *BaseExecutor) Cancel(ctx context.Context, localExecutionState store.LocalExecutionState) (err error) {
	execution := localExecutionState.Execution
	defer func() {
		if err != nil {
			e.handleFailure(ctx, localExecutionState, err, "Canceling")
		}
	}()

	log.Ctx(ctx).Debug().Str("Execution", execution.ID).Msg("Canceling execution")

	// TODO is returning the error the correct behaviour here?
	exe, err := e.executors.Get(ctx, execution.Job.Task().Engine.Type)
	if err != nil {
		return err
	}
	if err := exe.Cancel(ctx, execution.ID); err != nil {
		return err
	}

	e.callback.OnCancelComplete(ctx, CancelResult{
		ExecutionMetadata: NewExecutionMetadata(execution),
		RoutingMetadata: RoutingMetadata{
			SourcePeerID: e.ID,
			TargetPeerID: localExecutionState.RequesterNodeID,
		},
	})
	return err
}

func (e *BaseExecutor) handleFailure(ctx context.Context, localExecutionState store.LocalExecutionState, err error, operation string) {
	execution := localExecutionState.Execution
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
				TargetPeerID: localExecutionState.RequesterNodeID,
			},
			Err: err.Error(),
		})
	}
}

// compile-time interface check
var _ Executor = (*BaseExecutor)(nil)

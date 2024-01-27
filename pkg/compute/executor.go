package compute

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	wasmmodels "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/models"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

const StorageDirectoryPerms = 0755

type BaseExecutorParams struct {
	ID                     string
	Callback               Callback
	Store                  store.ExecutionStore
	Storages               storage.StorageProvider
	StorageDirectory       string
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
	Storages         storage.StorageProvider
	storageDirectory string
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
		storageDirectory: params.StorageDirectory,
		executors:        params.Executors,
		publishers:       params.Publishers,
		failureInjection: params.FailureInjectionConfig,
		resultsPath:      params.ResultsPath,
	}
}

func prepareInputVolumes(
	ctx context.Context,
	strgprovider storage.StorageProvider,
	storageDirectory string, inputSources ...*models.InputSource) (
	[]storage.PreparedStorage, func(context.Context) error, error) {
	inputVolumes, err := storage.ParallelPrepareStorage(ctx, strgprovider, storageDirectory, inputSources...)
	if err != nil {
		return nil, nil, err
	}
	return inputVolumes, func(ctx context.Context) error {
		return storage.ParallelCleanStorage(ctx, strgprovider, inputVolumes)
	}, nil
}

func prepareWasmVolumes(
	ctx context.Context,
	strgprovider storage.StorageProvider,
	storageDirectory string, wasmEngine wasmmodels.EngineSpec) (
	map[string][]storage.PreparedStorage, func(context.Context) error, error) {
	importModuleVolumes, err := storage.ParallelPrepareStorage(ctx, strgprovider, storageDirectory, wasmEngine.ImportModules...)
	if err != nil {
		return nil, nil, err
	}

	entryModuleVolumes, err := storage.ParallelPrepareStorage(ctx, strgprovider, storageDirectory, wasmEngine.EntryModule)
	if err != nil {
		return nil, nil, err
	}

	volumes := map[string][]storage.PreparedStorage{
		"importModules": importModuleVolumes,
		"entryModules":  entryModuleVolumes,
	}

	cleanup := func(ctx context.Context) error {
		err1 := storage.ParallelCleanStorage(ctx, strgprovider, importModuleVolumes)
		err2 := storage.ParallelCleanStorage(ctx, strgprovider, entryModuleVolumes)
		if err1 != nil || err2 != nil {
			return fmt.Errorf("Error cleaning up WASM volumes: %v, %v", err1, err2)
		}
		return nil
	}

	return volumes, cleanup, nil
}

// InputCleanupFn is a function type that defines the contract for cleaning up
// resources associated with input volume data after the job execution has either completed
// or failed to start. The function is expected to take a context.Context as an argument,
// which can be used for timeout and cancellation signals. It returns an error if
// the cleanup operation fails.
//
// For example, an InputCleanupFn might be responsible for deallocating storage used
// for input volumes, or deleting temporary input files that were created as part of the
// job's execution. The nature of it operation depends on the storage provided by `strgprovider` and
// input sources of the jobs associated tasks. For the case of a wasm job its input and entry module storage volumes
// should be removed via the method after the jobs execution reaches a terminal state.
type InputCleanupFn = func(context.Context) error

func PrepareRunArguments(
	ctx context.Context,
	strgprovider storage.StorageProvider,
	storageDirectory string,
	execution *models.Execution,
	resultsDir string,
) (*executor.RunCommandRequest, InputCleanupFn, error) {
	var cleanupFuncs []func(context.Context) error

	inputVolumes, inputCleanup, err := prepareInputVolumes(ctx, strgprovider, storageDirectory, execution.Job.Task().InputSources...)
	if err != nil {
		return nil, nil, err
	}
	cleanupFuncs = append(cleanupFuncs, inputCleanup)

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
	var engineArgs *models.SpecConfig
	if execution.Job.Task().Engine.IsType(models.EngineWasm) {
		wasmEngine, err := wasmmodels.DecodeSpec(execution.Job.Task().Engine)
		if err != nil {
			return nil, nil, err
		}

		volumes, wasmCleanup, err := prepareWasmVolumes(ctx, strgprovider, storageDirectory, wasmEngine)
		if err != nil {
			return nil, nil, err
		}

		cleanupFuncs = append(cleanupFuncs, wasmCleanup)

		engineArgs = &models.SpecConfig{
			Type:   models.EngineWasm,
			Params: wasmEngine.ToArguments(volumes["entryModules"][0], volumes["importModules"]...).ToMap(),
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
		}, func(ctx context.Context) error {
			log.Ctx(ctx).Info().Str("execution", execution.ID).Msg("cleaning up execution")
			cleanupErr := new(multierror.Error)
			for _, cleanupFunc := range cleanupFuncs {
				if err := cleanupFunc(ctx); err != nil {
					log.Ctx(ctx).Error().Err(err).Str("execution", execution.ID).Msg("cleaning up execution")
					cleanupErr = multierror.Append(cleanupErr, err)
				}
			}
			return cleanupErr.ErrorOrNil()
		}, nil
}

type StartResult struct {
	cleanup InputCleanupFn
	Err     error
}

func (r *StartResult) Cleanup(ctx context.Context) error {
	if r.cleanup != nil {
		return r.cleanup(ctx)
	}
	return nil
}

func (e *BaseExecutor) Start(ctx context.Context, execution *models.Execution) *StartResult {
	result := new(StartResult)
	jobExecutor, err := e.executors.Get(ctx, execution.Job.Task().Engine.Type)
	if err != nil {
		result.Err = fmt.Errorf("getting executor %s: %w", execution.Job.Task().Engine, err)
		return result
	}

	resultFolder, err := e.resultsPath.PrepareResultsDir(execution.ID)
	if err != nil {
		result.Err = fmt.Errorf("preparing results path: %w", err)
		return result
	}

	executionStorage := filepath.Join(e.storageDirectory, execution.JobID, execution.ID)
	if err := os.MkdirAll(executionStorage, StorageDirectoryPerms); err != nil {
		result.Err = fmt.Errorf("preparing storage path: %w", err)
		return result
	}

	args, cleanup, err := PrepareRunArguments(ctx, e.Storages, executionStorage, execution, resultFolder)
	result.cleanup = cleanup
	if err != nil {
		result.Err = fmt.Errorf("preparing arguments: %w", err)
		return result
	}

	if err := e.store.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID: execution.ID,
		ExpectedStates: []store.LocalExecutionStateType{
			store.ExecutionStateBidAccepted,
			store.ExecutionStateRunning, // allow retries during node restarts
		},
		NewState: store.ExecutionStateRunning,
	}); err != nil {
		result.Err = fmt.Errorf("updating execution state from expected: %s to: %s", store.ExecutionStateBidAccepted, store.ExecutionStateRunning)
		return result
	}

	log.Ctx(ctx).Debug().Msg("starting execution")

	if e.failureInjection.IsBadActor {
		result.Err = fmt.Errorf("i am a baaad node. i failed execution %s", execution.ID)
		return result
	}

	if err := jobExecutor.Start(ctx, args); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to start execution")
		result.Err = err
	}

	return result
}

func (e *BaseExecutor) Wait(ctx context.Context, state store.LocalExecutionState) (*models.RunCommandResult, error) {
	execution := state.Execution
	jobExecutor, err := e.executors.Get(ctx, execution.Job.Task().Engine.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to get executor %s: %w", execution.Job.Task().Engine, err)
	}

	waitC, errC := jobExecutor.Wait(ctx, execution.ID)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-waitC:
		return res, nil
	case err := <-errC:
		log.Ctx(ctx).Error().Err(err).Msg("failed to wait on execution")
		return nil, err
	}
}

// Run the execution after it has been accepted, and propose a result to the requester to be verified.
//
//nolint:funlen
func (e *BaseExecutor) Run(ctx context.Context, state store.LocalExecutionState) (err error) {
	execution := state.Execution
	ctx = log.Ctx(ctx).With().
		Str("job", execution.Job.ID).
		Str("execution", execution.ID).
		Logger().WithContext(ctx)

	stopwatch := telemetry.NewTimer(jobDurationMilliseconds)
	stopwatch.Start()
	operation := "Running"
	defer func() {
		if err != nil {
			e.handleFailure(ctx, state, err, operation)
		}
		stopwatch.Stop(ctx, state.Execution.Job.MetricAttributes()...)
	}()

	res := e.Start(ctx, execution)
	defer func() {
		if err := res.Cleanup(ctx); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to clean up start arguments")
		}
	}()
	if err := res.Err; err != nil {
		if errors.Is(err, executor.ErrAlreadyStarted) {
			// by not returning this error to the caller when the execution has already been started/is already running
			// we allow duplicate calls to `Run` to be idempotent and fall through to the below `Wait` call.
			log.Ctx(ctx).Warn().Err(err).Str("execution", execution.ID).
				Msg("execution is already running processing to wait on execution")
		} else {
			// We don't consider the job failed if the execution is already running or has already completed.
			// TODO(forrest): [correctness] do we really want to record a job failed metric if (one of) its execution(s)
			// failed to start? Perhaps it would be better to have metrics for execution failures here and job failures
			// higher up the call stack?
			jobsFailed.Add(ctx, 1)
			return err
		}
	}

	result, err := e.Wait(ctx, state)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			// TODO(forrest) [correctness]:
			// This is a special case for now. The ExecutorBuffer is using a context with a timeout to signal
			// an execution has timed out and should end. If we return an error here, the deferred handleFailure
			// call above will mark this execution as 'Failed'. Current testing and implementation expects executions
			// that have timed out to be in state 'Canceled', rather than 'Failed'. IMO An execution that doesn't
			// complete in the timeframe requested by a user should be 'Failed'. 'Canceled' probably ought to be
			// reserved for actions initiated by a user. We are ignoring _only_ context.DeadlineExceeded
			// errors and allowing context.Canceled errors be to returned so that when a compute node is shutdown
			// any active executions will be labeled as 'Failed' instead of canceled.
			// There is prior discussion regarding this point here:
			// https://github.com/bacalhau-project/bacalhau/pull/2705#discussion_r1283543457
			//
			// Moving forward we must avoid canceling executions via the context.Context. When pluggable executors
			// become the default since canceling the context will simply result in the RPC connection closing (I think)
			// The general solution here is to stop using contexts for canceling jobs and to instead make explicit calls
			// the an executors `Cancel` method.
			log.Ctx(ctx).Info().Msg("execution timeout exceeded canceling execution")
			return nil
		}
		return err
	}
	if result.ErrorMsg != "" {
		return fmt.Errorf("execution error: %s", result.ErrorMsg)
	}
	jobsCompleted.Add(ctx, 1)

	expectedState := store.ExecutionStateRunning
	publishedResult := models.SpecConfig{}

	// publish if the job has a publisher defined
	if !execution.Job.Task().Publisher.IsEmpty() {
		operation = "Publishing"
		if err := e.store.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
			ExecutionID:    execution.ID,
			ExpectedStates: []store.LocalExecutionStateType{expectedState},
			NewState:       store.ExecutionStatePublishing,
		}); err != nil {
			return err
		}

		expectedState = store.ExecutionStatePublishing

		resultsDir, err := e.resultsPath.EnsureResultsDir(state.Execution.ID)
		if err != nil {
			return err
		}

		defer func() {
			// cleanup resources
			log.Ctx(ctx).Debug().Msgf("Cleaning up result folder for %s: %s", execution.ID, resultsDir)
			err = os.RemoveAll(resultsDir)
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Msgf("failed to remove results folder at %s", resultsDir)
			}
		}()

		publishedResult, err = e.publish(ctx, state, resultsDir)
		if err != nil {
			return err
		}
	}

	// mark the execution as completed
	if err := e.store.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:    execution.ID,
		ExpectedStates: []store.LocalExecutionStateType{expectedState},
		NewState:       store.ExecutionStateCompleted,
	}); err != nil {
		return err
	}

	// notify requester
	e.callback.OnRunComplete(ctx, RunResult{
		ExecutionMetadata: NewExecutionMetadata(execution),
		RoutingMetadata: RoutingMetadata{
			SourcePeerID: e.ID,
			TargetPeerID: state.RequesterNodeID,
		},
		PublishResult:    &publishedResult,
		RunCommandResult: result,
	})
	return err
}

// Publish the result of an execution after it has been verified.
func (e *BaseExecutor) publish(ctx context.Context, localExecutionState store.LocalExecutionState,
	resultFolder string) (publishedResult models.SpecConfig, err error) {
	execution := localExecutionState.Execution
	log.Ctx(ctx).Debug().Msgf("Publishing execution %s", execution.ID)

	jobPublisher, err := e.publishers.Get(ctx, execution.Job.Task().Publisher.Type)
	if err != nil {
		err = fmt.Errorf("failed to get publisher %s: %w", execution.Job.Task().Publisher.Type, err)
		return
	}
	publishedResult, err = jobPublisher.PublishResult(ctx, execution, resultFolder)
	if err != nil {
		err = fmt.Errorf("failed to publish result: %w", err)
		return
	}

	log.Ctx(ctx).Debug().
		Str("execution", execution.ID).
		Msg("Execution published")

	return
}

// Cancel the execution.
func (e *BaseExecutor) Cancel(ctx context.Context, state store.LocalExecutionState) (err error) {
	execution := state.Execution
	defer func() {
		if err != nil {
			e.handleFailure(ctx, state, err, "Canceling")
		}
	}()

	log.Ctx(ctx).Debug().Str("Execution", execution.ID).Msg("Canceling execution")

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
			TargetPeerID: state.RequesterNodeID,
		},
	})
	return err
}

func (e *BaseExecutor) handleFailure(ctx context.Context, state store.LocalExecutionState, err error, operation string) {
	execution := state.Execution
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
				TargetPeerID: state.RequesterNodeID,
			},
			Err: err.Error(),
		})
	}
}

// compile-time interface check
var _ Executor = (*BaseExecutor)(nil)

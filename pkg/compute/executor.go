package compute

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	wasmmodels "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/models"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

const StorageDirectoryPerms = 0o755

type BaseExecutorParams struct {
	ID                     string
	Store                  store.ExecutionStore
	Storages               storage.StorageProvider
	StorageDirectory       string
	Executors              executor.ExecProvider
	ResultsPath            ResultsPath
	Publishers             publisher.PublisherProvider
	FailureInjectionConfig models.FailureInjectionConfig
}

// BaseExecutor is the base implementation for backend service.
// All operations are executed asynchronously, and a callback is used to notify the caller of the result.
type BaseExecutor struct {
	ID               string
	store            store.ExecutionStore
	Storages         storage.StorageProvider
	storageDirectory string
	executors        executor.ExecProvider
	publishers       publisher.PublisherProvider
	resultsPath      ResultsPath
	failureInjection models.FailureInjectionConfig
}

func NewBaseExecutor(params BaseExecutorParams) *BaseExecutor {
	return &BaseExecutor{
		ID:               params.ID,
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
	storageProvider storage.StorageProvider,
	storageDirectory string, inputSources ...*models.InputSource) (
	[]storage.PreparedStorage, func(context.Context) error, error,
) {
	inputVolumes, err := storage.ParallelPrepareStorage(ctx, storageProvider, storageDirectory, inputSources...)
	if err != nil {
		return nil, nil, err
	}
	return inputVolumes, func(ctx context.Context) error {
		return storage.ParallelCleanStorage(ctx, storageProvider, inputVolumes)
	}, nil
}

func prepareWasmVolumes(
	ctx context.Context,
	storageProvider storage.StorageProvider,
	storageDirectory string, wasmEngine wasmmodels.EngineSpec) (
	map[string][]storage.PreparedStorage, func(context.Context) error, error,
) {
	importModuleVolumes, err := storage.ParallelPrepareStorage(ctx, storageProvider, storageDirectory, wasmEngine.ImportModules...)
	if err != nil {
		return nil, nil, err
	}

	entryModuleVolumes, err := storage.ParallelPrepareStorage(ctx, storageProvider, storageDirectory, wasmEngine.EntryModule)
	if err != nil {
		return nil, nil, err
	}

	volumes := map[string][]storage.PreparedStorage{
		"importModules": importModuleVolumes,
		"entryModules":  entryModuleVolumes,
	}

	cleanup := func(ctx context.Context) error {
		err1 := storage.ParallelCleanStorage(ctx, storageProvider, importModuleVolumes)
		err2 := storage.ParallelCleanStorage(ctx, storageProvider, entryModuleVolumes)
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
// job's execution. The nature of it operation depends on the storage provided by `storageProvider` and
// input sources of the jobs associated tasks. For the case of a wasm job its input and entry module storage volumes
// should be removed via the method after the jobs execution reaches a terminal state.
type InputCleanupFn = func(context.Context) error

func PrepareRunArguments(
	ctx context.Context,
	storageProvider storage.StorageProvider,
	storageDirectory string,
	execution *models.Execution,
	resultsDir string,
) (*executor.RunCommandRequest, InputCleanupFn, error) {
	var cleanupFuncs []func(context.Context) error

	inputVolumes, inputCleanup, err := prepareInputVolumes(ctx, storageProvider, storageDirectory, execution.Job.Task().InputSources...)
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

		volumes, wasmCleanup, err := prepareWasmVolumes(ctx, storageProvider, storageDirectory, wasmEngine)
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
			var cleanupErr error
			for _, cleanupFunc := range cleanupFuncs {
				if err := cleanupFunc(ctx); err != nil {
					log.Ctx(ctx).Error().Err(err).Str("execution", execution.ID).Msg("cleaning up execution")
					cleanupErr = errors.Join(cleanupErr, err)
				}
			}
			return cleanupErr
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

	if err = e.store.UpdateExecutionState(ctx, store.UpdateExecutionRequest{
		ExecutionID: execution.ID,
		Condition: store.UpdateExecutionCondition{
			ExpectedStates: []models.ExecutionStateType{
				models.ExecutionStateBidAccepted,
				models.ExecutionStateRunning, // allow retries during node restarts
			},
		},
		NewValues: models.Execution{
			ComputeState: models.NewExecutionState(models.ExecutionStateRunning),
		},
	}); err != nil {
		result.Err = fmt.Errorf("updating execution state from expected: %s to: %s",
			models.ExecutionStateBidAccepted, models.ExecutionStateRunning)
		return result
	}

	log.Ctx(ctx).Debug().Msg("starting execution")

	if e.failureInjection.IsBadActor {
		result.Err = fmt.Errorf("i am a bad node. i failed execution %s", execution.ID)
		return result
	}

	if err := jobExecutor.Start(ctx, args); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to start execution")
		result.Err = err
	}

	return result
}

func (e *BaseExecutor) Wait(ctx context.Context, execution *models.Execution) (*models.RunCommandResult, error) {
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
func (e *BaseExecutor) Run(ctx context.Context, execution *models.Execution) (err error) {
	ctx = log.Ctx(ctx).With().
		Str("job", execution.Job.ID).
		Str("execution", execution.ID).
		Logger().WithContext(ctx)

	stopwatch := telemetry.Timer(ctx, jobDurationMilliseconds, execution.Job.MetricAttributes()...)
	topic := EventTopicExecutionRunning
	defer func() {
		if err != nil {
			if !bacerrors.IsErrorWithCode(err, executor.ExecutionAlreadyCancelled) {
				e.handleFailure(ctx, execution, err, topic)
			}
		}
		dur := stopwatch()
		log.Ctx(ctx).Debug().
			Dur("duration", dur).
			Str("jobID", execution.JobID).
			Str("executionID", execution.ID).
			Msg("run complete")
	}()

	res := e.Start(ctx, execution)
	defer func() {
		if err := res.Cleanup(ctx); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to clean up start arguments")
		}
	}()
	if err := res.Err; err != nil {
		if bacerrors.IsErrorWithCode(err, executor.ExecutionAlreadyStarted) {
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

	result, err := e.Wait(ctx, execution)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			// TODO(forrest) [correctness]:
			// The ExecutorBuffer is using a context with a timeout to signal an execution has timed out and should end.
			//
			// We don't handle context.Canceled here as it means the node is shutting down. Still we should do a
			// better job at gracefully shutting down the execution and either reporting that to the requester
			// or retrying the execution during startup.
			//
			// Moving forward we must avoid canceling executions via the context.Context. When pluggable executors
			// become the default since canceling the context will simply result in the RPC connection closing (I think)
			// The general solution here is to stop using contexts for canceling jobs and to instead make explicit calls
			// the an executors `Cancel` method.
			return NewErrExecTimeout(execution.Job.Task().Timeouts.GetExecutionTimeout())
		}
		return err
	}
	if result.ErrorMsg != "" {
		return fmt.Errorf("%s", result.ErrorMsg)
	}
	jobsCompleted.Add(ctx, 1)

	expectedState := models.ExecutionStateRunning
	var publishedResult *models.SpecConfig

	// publish if the job has a publisher defined
	if !execution.Job.Task().Publisher.IsEmpty() {
		topic = EventTopicExecutionPublishing
		if err = e.store.UpdateExecutionState(ctx, store.UpdateExecutionRequest{
			ExecutionID: execution.ID,
			Condition: store.UpdateExecutionCondition{
				ExpectedStates: []models.ExecutionStateType{expectedState},
			},
			NewValues: models.Execution{
				ComputeState: models.NewExecutionState(models.ExecutionStatePublishing),
				RunOutput:    result,
			},
		}); err != nil {
			return err
		}

		expectedState = models.ExecutionStatePublishing

		resultsDir, err := e.resultsPath.EnsureResultsDir(execution.ID)
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

		publishedResult, err = e.publish(ctx, execution, resultsDir)
		if err != nil {
			return err
		}
	}

	// mark the execution as completed
	if err = e.store.UpdateExecutionState(ctx, store.UpdateExecutionRequest{
		ExecutionID: execution.ID,
		Condition: store.UpdateExecutionCondition{
			ExpectedStates: []models.ExecutionStateType{expectedState},
		},
		NewValues: models.Execution{
			ComputeState:    models.NewExecutionState(models.ExecutionStateCompleted),
			PublishedResult: publishedResult,
			RunOutput:       result,
		},
		Events: []models.Event{*ExecCompletedEvent()},
	}); err != nil {
		return err
	}

	return err
}

// Publish the result of an execution after it has been verified.
func (e *BaseExecutor) publish(ctx context.Context, execution *models.Execution,
	resultFolder string,
) (*models.SpecConfig, error) {
	log.Ctx(ctx).Debug().Msgf("Publishing execution %s", execution.ID)

	jobPublisher, err := e.publishers.Get(ctx, execution.Job.Task().Publisher.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to get publisher %s: %w", execution.Job.Task().Publisher.Type, err)
	}
	publishedResult, err := jobPublisher.PublishResult(ctx, execution, resultFolder)
	if err != nil {
		return nil, bacerrors.Wrap(err, "failed to publish result")
	}
	log.Ctx(ctx).Debug().
		Str("execution", execution.ID).
		Msg("Execution published")

	return &publishedResult, nil
}

// Cancel the execution.
func (e *BaseExecutor) Cancel(ctx context.Context, execution *models.Execution) error {
	log.Ctx(ctx).Debug().Str("Execution", execution.ID).Msg("Canceling execution")
	exe, err := e.executors.Get(ctx, execution.Job.Task().Engine.Type)
	if err != nil {
		return err
	}
	return exe.Cancel(ctx, execution.ID)
}

func (e *BaseExecutor) handleFailure(ctx context.Context, execution *models.Execution, err error, topic models.EventTopic) {
	log.Ctx(ctx).Warn().Err(err).Msgf("%s failed", topic)

	updateError := e.store.UpdateExecutionState(ctx, store.UpdateExecutionRequest{
		ExecutionID: execution.ID,
		NewValues: models.Execution{
			ComputeState: models.NewExecutionState(models.ExecutionStateFailed).WithMessage(err.Error()),
		},
		Events: []models.Event{*models.NewEvent(topic).WithError(err)},
	})

	if updateError != nil {
		log.Ctx(ctx).Error().Err(updateError).Msgf("Failed to update execution (%s) state to failed: %s", execution.ID, updateError)
	}
}

// compile-time interface check
var _ Executor = (*BaseExecutor)(nil)

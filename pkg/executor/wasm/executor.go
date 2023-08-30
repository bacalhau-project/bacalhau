package wasm

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/tetratelabs/wazero"
	"go.uber.org/atomic"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/system"

	"github.com/bacalhau-project/bacalhau/pkg/lib/math"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/resource"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	wasmmodels "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/models"
	wasmlogs "github.com/bacalhau-project/bacalhau/pkg/logger/wasm"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/bacalhau-project/bacalhau/pkg/util/filefs"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
	"github.com/bacalhau-project/bacalhau/pkg/util/mountfs"
	"github.com/bacalhau-project/bacalhau/pkg/util/touchfs"
)

type Executor struct {
	handlers generic.SyncMap[string, *executionHandler]
}

func NewExecutor() (*Executor, error) {
	return &Executor{}, nil
}

func (e *Executor) IsInstalled(context.Context) (bool, error) {
	// WASM executor runs natively in Go and so is always available
	return true, nil
}

func (*Executor) ShouldBid(ctx context.Context, request bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
	return semantic.NewChainedSemanticBidStrategy().ShouldBid(ctx, request)
}

func (*Executor) ShouldBidBasedOnUsage(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest,
	usage models.Resources,
) (bidstrategy.BidStrategyResponse, error) {
	return resource.NewChainedResourceBidStrategy().ShouldBidBasedOnUsage(ctx, request, usage)
}

func (e *Executor) Start(ctx context.Context, request *executor.RunCommandRequest) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/executor/wasm.Executor.Start")
	defer span.End()

	if handler, found := e.handlers.Get(request.ExecutionID); found {
		if handler.active() {
			// TODO we can make this method idempotent by not returning an error here if the execution is already active
			return fmt.Errorf("execution (%s) already started", request.ExecutionID)
		} else {
			// TODO what should we do if an execution has already completed and we try to start it again? Just rerun it?
			return fmt.Errorf("execution (%s) already completed", request.ExecutionID)
		}
	}

	engineParams, err := wasmmodels.DecodeArguments(request.EngineParams)
	if err != nil {
		return fmt.Errorf("decoding wasm arguments: %w", err)
	}

	// Apply memory limits to the runtime. We have to do this in multiples of
	// the WASM page size of 64kb, so round up to the nearest page size if the
	// limit is not specified as a multiple of that.
	engineConfig := wazero.NewRuntimeConfig().WithCloseOnContextDone(true)
	if request.Resources.Memory > 0 {
		const pageSize = 65536
		pageLimit := request.Resources.Memory/pageSize + math.Min(request.Resources.Memory%pageSize, 1)
		engineConfig = engineConfig.WithMemoryLimitPages(uint32(pageLimit))
	}

	rootFs, err := e.makeFsFromStorage(ctx, request.ResultsDir, request.Inputs, request.Outputs)
	if err != nil {
		return err
	}

	// Create a new log manager and obtain some writers that we can pass to the wasm
	// configuration
	wasmLogs, err := wasmlogs.NewLogManager(ctx, request.ExecutionID)
	if err != nil {
		return err
	}

	handler := &executionHandler{
		runtime:    wazero.NewRuntimeWithConfig(ctx, engineConfig),
		arguments:  engineParams,
		fs:         rootFs,
		inputs:     request.Inputs,
		resultsDir: request.ResultsDir,
		limits:     request.OutputLimits,
		logger: log.With().
			Str("execution", request.ExecutionID).
			Str("job", request.JobID).
			Str("entrypoint", engineParams.EntryPoint).
			Logger(),
		logManager: wasmLogs,
		activeCh:   make(chan bool),
		waitCh:     make(chan bool),
		running:    atomic.NewBool(false),
	}

	// register the handler for this executionID
	e.handlers.Put(request.ExecutionID, handler)
	go handler.run(ctx)
	return nil
}

func (e *Executor) Wait(ctx context.Context, executionID string) (<-chan *models.RunCommandResult, error) {
	handler, found := e.handlers.Get(executionID)
	if !found {
		return nil, fmt.Errorf("execution (%s) not found", executionID)
	}
	ch := make(chan *models.RunCommandResult)
	go e.doWait(ctx, ch, handler)
	return ch, nil
}

func (e *Executor) doWait(ctx context.Context, out chan *models.RunCommandResult, handle *executionHandler) {
	defer close(out)
	select {
	/*
		case <-ctx.Done():
			out <- &models.RunCommandResult{ErrorMsg: ctx.Err().Error()}

	*/
	case <-handle.waitCh:
		// FIXME: don't return an error from this method and instead populate the error from the returned structure,
		// which the method already does internally.
		res, err := executor.WriteJobResults(
			handle.resultsDir,
			handle.result.stdOut,
			handle.result.stdErr,
			int(handle.result.exitcode),
			handle.result.err,
			handle.limits,
		)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to write job results TODO FIX ME")
		}
		out <- res
	}
}

func (e *Executor) Run(
	ctx context.Context,
	request *executor.RunCommandRequest,
) (*models.RunCommandResult, error) {
	if err := e.Start(ctx, request); err != nil {
		return nil, err
	}
	res, err := e.Wait(ctx, request.ExecutionID)
	if err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case out := <-res:
		return out, nil
	}
}

func (e *Executor) Cancel(ctx context.Context, executionID string) error {
	handler, found := e.handlers.Get(executionID)
	if !found {
		return fmt.Errorf("execution (%s) not found", executionID)
	}
	return handler.kill(ctx)
}

func (e *Executor) GetOutputStream(ctx context.Context, executionID string, withHistory bool, follow bool) (io.ReadCloser, error) {
	handler, found := e.handlers.Get(executionID)
	if !found {
		return nil, fmt.Errorf("execution (%s) not found", executionID)
	}
	return handler.outputStream(ctx, withHistory, follow)
}

// makeFsFromStorage sets up a virtual filesystem (represented by an fs.FS) that
// will be the filesystem exposed to our WASM. The strategy for this is to:
//
//   - mount each input at the name specified by Path
//   - make a directory in the job results directory for each output and mount that
//     at the name specified by Name
func (e *Executor) makeFsFromStorage(
	ctx context.Context,
	jobResultsDir string,
	volumes []storage.PreparedStorage,
	outputs []*models.ResultPath) (fs.FS, error) {
	var err error
	rootFs := mountfs.New()

	for _, v := range volumes {
		log.Ctx(ctx).Debug().
			Str("input", v.InputSource.Target).
			Str("source", v.Volume.Source).
			Msg("Using input")

		var stat os.FileInfo
		stat, err = os.Stat(v.Volume.Source)
		if err != nil {
			return nil, err
		}

		var inputFs fs.FS
		if stat.IsDir() {
			inputFs = os.DirFS(v.Volume.Source)
		} else {
			inputFs = filefs.New(v.Volume.Source)
		}

		err = rootFs.Mount(v.InputSource.Target, inputFs)
		if err != nil {
			return nil, err
		}
	}

	for _, output := range outputs {
		if output.Name == "" {
			return nil, fmt.Errorf("output volume has no name: %+v", output)
		}

		if output.Path == "" {
			return nil, fmt.Errorf("output volume has no path: %+v", output)
		}

		srcd := filepath.Join(jobResultsDir, output.Name)
		log.Ctx(ctx).Debug().
			Str("output", output.Name).
			Str("dir", srcd).
			Msg("Collecting output")

		err = os.Mkdir(srcd, util.OS_ALL_R|util.OS_ALL_X|util.OS_USER_W)
		if err != nil {
			return nil, err
		}

		err = rootFs.Mount(output.Name, touchfs.New(srcd))
		if err != nil {
			return nil, err
		}
	}

	return rootFs, nil
}

// Compile-time check that Executor implements the Executor interface.
var _ executor.Executor = (*Executor)(nil)

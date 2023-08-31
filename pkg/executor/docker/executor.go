package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.uber.org/atomic"

	dockermodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	pkgUtil "github.com/bacalhau-project/bacalhau/pkg/util"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/resource"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/docker/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
)

const NanoCPUCoefficient = 1000000000

const (
	labelExecutorName = "bacalhau-executor"
	labelJobName      = "bacalhau-jobID"
	labelExecutionID  = "bacalhau-executionID"
)

type Executor struct {
	// used to allow multiple docker executors to run against the same docker server
	ID string

	handlers generic.SyncMap[string, *executionHandler]

	activeFlags map[string]chan struct{}
	complete    map[string]chan struct{}
	client      *docker.Client
	results     generic.SyncMap[string, *models.RunCommandResult]
}

func NewExecutor(
	_ context.Context,
	id string,
) (*Executor, error) {
	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		return nil, err
	}

	de := &Executor{
		ID:          id,
		client:      dockerClient,
		activeFlags: make(map[string]chan struct{}),
		complete:    make(map[string]chan struct{}),
	}

	return de, nil
}

func (e *Executor) Shutdown(ctx context.Context) error {
	// We have to use a detached context, rather than the one passed in to `NewExecutor`, as it may have already been
	// canceled and so would prevent us from performing any cleanup work.
	safeCtx := pkgUtil.NewDetachedContext(ctx)
	if config.ShouldKeepStack() || !e.client.IsInstalled(safeCtx) {
		return nil
	}

	err := e.client.RemoveObjectsWithLabel(safeCtx, labelExecutorName, e.ID)
	logLevel := map[bool]zerolog.Level{true: zerolog.DebugLevel, false: zerolog.ErrorLevel}[err == nil]
	log.Ctx(ctx).WithLevel(logLevel).Err(err).Msg("Cleaned up all Docker resources")

	return nil
}

// IsInstalled checks if docker itself is installed.
func (e *Executor) IsInstalled(ctx context.Context) (bool, error) {
	return e.client.IsInstalled(ctx), nil
}

func (e *Executor) ShouldBid(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest,
) (bidstrategy.BidStrategyResponse, error) {
	return semantic.NewImagePlatformBidStrategy(e.client).ShouldBid(ctx, request)
}

func (e *Executor) ShouldBidBasedOnUsage(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest,
	usage models.Resources,
) (bidstrategy.BidStrategyResponse, error) {
	// TODO(forrest): should this just return true always?
	return resource.NewChainedResourceBidStrategy().ShouldBidBasedOnUsage(ctx, request, usage)
}

//nolint:funlen,gocyclo // will clean up
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

// TODO need a better name as this starts an execution for the request, not the actual executor this method is on.
// Launc, Exec, Dispatch?
func (e *Executor) Start(ctx context.Context, request *executor.RunCommandRequest) error {
	log.Ctx(ctx).Info().
		Str("executionID", request.ExecutionID).
		Str("jobID", request.JobID).
		Msg("starting execution")

	if handler, found := e.handlers.Get(request.ExecutionID); found {
		if handler.active() {
			// TODO we can make this method idempotent by not returning an error here if the execution is already active
			return fmt.Errorf("execution (%s) already started", request.ExecutionID)
		} else {
			// TODO what should we do if an execution has already completed and we try to start it again? Just rerun it?
			return fmt.Errorf("execution (%s) already completed", request.ExecutionID)
		}
	}

	jobContainer, err := e.newDockerJobContainer(ctx, &dockerJobContainerParams{
		ExecutionID:   request.ExecutionID,
		JobID:         request.JobID,
		EngineSpec:    request.EngineParams,
		NetworkConfig: request.Network,
		Resources:     request.Resources,
		Inputs:        request.Inputs,
		Outputs:       request.Outputs,
		ResultsDir:    request.ResultsDir,
	})
	if err != nil {
		return fmt.Errorf("failed to create docker job container: %w", err)
	}

	handler := &executionHandler{
		client: e.client,
		logger: log.With().
			Str("container", jobContainer.ID).
			Str("execution", request.ExecutionID).
			Str("job", request.JobID).
			Logger(),
		ID:          e.ID,
		executionID: request.ExecutionID,
		containerID: jobContainer.ID,
		resultsDir:  request.ResultsDir,
		limits:      request.OutputLimits,
		keepStack:   config.ShouldKeepStack(),
		waitCh:      make(chan bool),
		activeCh:    make(chan bool),
		running:     atomic.NewBool(false),
	}

	// register the handler for this executionID
	e.handlers.Put(request.ExecutionID, handler)
	// run the container.
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
	log.Info().Str("executionID", handle.executionID).Msg("waiting on execution")
	defer close(out)
	select {
	case <-ctx.Done():
		out <- executor.NewFailedResult(fmt.Sprintf("context canceled while waiting for execution: %s", ctx.Err()))
	case <-handle.waitCh:
		log.Info().Str("executionID", handle.executionID).Msg("received results from execution")
		out <- handle.result
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
	return handler.outputStream(ctx, withHistory, found)
}

type dockerJobContainerParams struct {
	ExecutionID   string
	JobID         string
	EngineSpec    *models.SpecConfig
	NetworkConfig *models.NetworkConfig
	Resources     *models.Resources
	Inputs        []storage.PreparedStorage
	Outputs       []*models.ResultPath
	ResultsDir    string
}

func (e *Executor) newDockerJobContainer(ctx context.Context, params *dockerJobContainerParams) (container.CreateResponse, error) {
	// decode the request arguments, bail if they are invalid.
	dockerArgs, err := dockermodels.DecodeSpec(params.EngineSpec)
	if err != nil {
		return container.CreateResponse{}, fmt.Errorf("decoding engine spec: %w", err)
	}
	containerConfig := &container.Config{
		Image:      dockerArgs.Image,
		Tty:        false,
		Env:        dockerArgs.EnvironmentVariables,
		Entrypoint: dockerArgs.Entrypoint,
		Cmd:        dockerArgs.Parameters,
		Labels:     e.containerLabels(params.ExecutionID, params.JobID),
		WorkingDir: dockerArgs.WorkingDirectory,
	}

	mounts, err := makeContainerMounts(ctx, params.Inputs, params.Outputs, params.ResultsDir)
	if err != nil {
		return container.CreateResponse{}, fmt.Errorf("creating container mounts: %w", err)
	}

	// Create GPU request if the job requests it
	// TODO we need to use the resource units requested by for the GPU.
	var deviceRequests []container.DeviceRequest
	if params.Resources.GPU > 0 {
		deviceRequests = append(deviceRequests,
			container.DeviceRequest{
				DeviceIDs:    []string{"0"}, // TODO: how do we know which device ID to use?
				Capabilities: [][]string{{"gpu"}},
			},
		)
		log.Ctx(ctx).Trace().Msgf("Adding %d GPUs to request", params.Resources.GPU)
	}

	hostConfig := &container.HostConfig{
		Mounts: mounts,
		Resources: container.Resources{
			Memory:         int64(params.Resources.Memory),
			NanoCPUs:       int64(params.Resources.CPU * NanoCPUCoefficient),
			DeviceRequests: deviceRequests,
		},
	}

	if _, set := os.LookupEnv("SKIP_IMAGE_PULL"); !set {
		dockerCreds := config.GetDockerCredentials()
		if pullErr := e.client.PullImage(ctx, dockerArgs.Image, dockerCreds); pullErr != nil {
			pullErr = errors.Wrapf(pullErr, docker.ImagePullError, dockerArgs.Image)
			return container.CreateResponse{}, fmt.Errorf("failed to pull docker image: %w", pullErr)
		}
	}
	log.Ctx(ctx).Trace().Msgf("Container: %+v %+v", containerConfig, mounts)
	// Create a network if the job requests it, modifying the containerConfig and hostConfig.
	err = e.setupNetworkForJob(ctx, params.JobID, params.ExecutionID, params.NetworkConfig, containerConfig, hostConfig)
	if err != nil {
		return container.CreateResponse{}, fmt.Errorf("setting up network: %w", err)
	}

	// create the docker container (but don't start it)
	jobContainer, err := e.client.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		nil,
		nil,
		e.containerName(params.ExecutionID, params.JobID),
	)
	if err != nil {
		return container.CreateResponse{}, fmt.Errorf("creating container: %w", err)
	}
	return jobContainer, nil
}

func makeContainerMounts(ctx context.Context, inputs []storage.PreparedStorage, outputs []*models.ResultPath, resultsDir string) ([]mount.Mount, error) {
	// the actual mounts we will give to the container
	// these are paths for both input and output data
	var mounts []mount.Mount
	for _, input := range inputs {
		if input.Volume.Type == storage.StorageVolumeConnectorBind {
			log.Ctx(ctx).Trace().Msgf("Input Volume: %+v %+v", input.InputSource, input.Volume)

			mounts = append(mounts, mount.Mount{
				Type:     mount.TypeBind,
				ReadOnly: input.Volume.ReadOnly,
				Source:   input.Volume.Source,
				Target:   input.Volume.Target,
			})
		} else {
			return nil, fmt.Errorf("unknown storage volume type: %s", input.Volume.Type)
		}
	}

	// for this phase of the outputs we ignore the engine because it's just about collecting the
	// data from the job and keeping it locally
	// the engine property of the output storage spec is how we will "publish" the output volume
	// if and when the deal is settled
	for _, output := range outputs {
		if output.Name == "" {
			return nil, fmt.Errorf("output volume has no name: %+v", output)
		}

		if output.Path == "" {
			return nil, fmt.Errorf("output volume has no Location: %+v", output)
		}

		srcd := filepath.Join(resultsDir, output.Name)
		if err := os.Mkdir(srcd, util.OS_ALL_R|util.OS_ALL_X|util.OS_USER_W); err != nil {
			return nil, fmt.Errorf("failed to create results dir for execution: %w", err)

		}

		log.Ctx(ctx).Trace().Msgf("Output Volume: %+v", output)

		// create a mount so the output data does not need to be copied back to the host
		mounts = append(mounts, mount.Mount{
			Type: mount.TypeBind,
			// this is an output volume so can be written to
			ReadOnly: false,
			// we create a named folder in the job results folder for this output
			Source: srcd,
			// the path of the output volume is from the perspective of inside the container
			Target: output.Path,
		})
	}
	return mounts, nil
}

func (e *Executor) dockerObjectName(executionID string, jobID string, parts ...string) string {
	strs := []string{"bacalhau", e.ID, jobID, executionID}
	strs = append(strs, parts...)
	return strings.Join(strs, "-")
}

func (e *Executor) containerName(executionID string, jobID string) string {
	return e.dockerObjectName(executionID, jobID, "executor")
}

func (e *Executor) containerLabels(executionID, jobID string) map[string]string {
	return map[string]string{
		labelExecutorName: e.ID,
		labelJobName:      e.labelJobValue(jobID),
		labelExecutionID:  labelExecutionValue(e.ID, executionID),
	}
}

func (e *Executor) labelJobValue(jobID string) string {
	return e.ID + jobID
}

func labelExecutionValue(executorID string, executionID string) string {
	return executorID + executionID
}

// Compile-time interface check:
var _ executor.Executor = (*Executor)(nil)

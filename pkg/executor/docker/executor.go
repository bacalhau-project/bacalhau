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
	"github.com/samber/lo"
	"go.uber.org/atomic"

	dockermodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	pkgUtil "github.com/bacalhau-project/bacalhau/pkg/util"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
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

	// handlers is a map of executionID to its handler.
	handlers generic.SyncMap[string, *executionHandler]

	activeFlags map[string]chan struct{}
	complete    map[string]chan struct{}
	client      *docker.Client
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
	return bidstrategy.NewBidResponse(true, "not place additional requirements on Docker jobs"), nil
}

// Start initiates an execution based on the provided RunCommandRequest.
func (e *Executor) Start(ctx context.Context, request *executor.RunCommandRequest) error {
	log.Ctx(ctx).Info().
		Str("executionID", request.ExecutionID).
		Str("jobID", request.JobID).
		Msg("starting execution")

	// It's possible that this is being called due to a restart. Whilst we check the handlers to see
	// if we already have a running execution, this map will be empty on a compute node restart. As
	// a result we need to explicitly ask docker if there is a running container with the relevant
	// bacalhau execution label _before_ we do anything else.  If we are able to find one then we
	// will use that container in the executionHandler that we create.
	containerID, err := e.FindRunningContainer(ctx, request.ExecutionID)

	if err != nil {
		// Unable to find a running container for this execution, we will instead check for a handler, and
		// failing that will create a new containe.
		if handler, found := e.handlers.Get(request.ExecutionID); found {
			if handler.active() {
				return fmt.Errorf("starting execution (%s): %w", request.ExecutionID, executor.ErrAlreadyStarted)
			} else {
				return fmt.Errorf("starting execution (%s): %w", request.ExecutionID, executor.ErrAlreadyComplete)
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

		containerID = jobContainer.ID
	}

	handler := &executionHandler{
		client: e.client,
		logger: log.With().
			Str("container", containerID).
			Str("execution", request.ExecutionID).
			Str("job", request.JobID).
			Logger(),
		ID:          e.ID,
		executionID: request.ExecutionID,
		containerID: containerID,
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

// Wait initiates a wait for the completion of a specific execution using its
// executionID. The function returns two channels: one for the result and another
// for any potential error. If the executionID is not found, an error is immediately
// sent to the error channel. Otherwise, an internal goroutine (doWait) is spawned
// to handle the asynchronous waiting. Callers should use the two returned channels
// to wait for the result of the execution or an error. This can be due to issues
// either beginning the wait or in getting the response. This approach allows the
// caller to synchronize Wait with calls to Start, waiting for the execution to complete.
func (e *Executor) Wait(ctx context.Context, executionID string) (<-chan *models.RunCommandResult, <-chan error) {
	handler, found := e.handlers.Get(executionID)
	resultCh := make(chan *models.RunCommandResult, 1)
	errCh := make(chan error, 1)

	if !found {
		errCh <- fmt.Errorf("waiting on execution (%s): %w", executionID, executor.ErrNotFound)
		return resultCh, errCh
	}

	go e.doWait(ctx, resultCh, errCh, handler)
	return resultCh, errCh
}

// doWait is a helper function that actively waits for an execution to finish. It
// listens on the executionHandler's wait channel for completion signals. Once the
// signal is received, the result is sent to the provided output channel. If there's
// a cancellation request (context is done) before completion, an error is relayed to
// the error channel. If the execution result is nil, an error suggests a potential
// flaw in the executor logic.
func (e *Executor) doWait(ctx context.Context, out chan *models.RunCommandResult, errCh chan error, handle *executionHandler) {
	log.Info().Str("executionID", handle.executionID).Msg("waiting on execution")
	defer close(out)
	defer close(errCh)

	select {
	case <-ctx.Done():
		errCh <- ctx.Err() // Send the cancellation error to the error channel
		return
	case <-handle.waitCh:
		if handle.result != nil {
			log.Info().Str("executionID", handle.executionID).Msg("received results from execution")
			out <- handle.result
		} else {
			// NB(forrest): this shouldn't happen with the wasm and docker executors, but handling it as it
			// represents a significant error in executor logic, which may occur in future pluggable executor impls.
			errCh <- fmt.Errorf("execution (%s) result is nil", handle.executionID)
		}
	}
}

// Cancel tries to cancel a specific execution by its executionID.
// It returns an error if the execution is not found.
func (e *Executor) Cancel(ctx context.Context, executionID string) error {
	handler, found := e.handlers.Get(executionID)
	if !found {
		return fmt.Errorf("canceling execution (%s): %w", executionID, executor.ErrNotFound)
	}
	return handler.kill(ctx)
}

// GetOutputStream provides a stream of output logs for a specific execution.
// Parameters 'withHistory' and 'follow' control whether to include past logs
// and whether to keep the stream open for new logs, respectively.
// It returns an error if the execution is not found.
func (e *Executor) GetOutputStream(ctx context.Context, executionID string, withHistory bool, follow bool) (io.ReadCloser, error) {
	handler, found := e.handlers.Get(executionID)
	if !found {
		return nil, fmt.Errorf("getting outputs for execution (%s): %w", executionID, executor.ErrNotFound)
	}
	return handler.outputStream(ctx, withHistory, found)
}

// Run initiates and waits for the completion of an execution in one call.
// This method serves as a higher-level convenience function that
// internally calls Start and Wait methods.
// It returns the result of the execution or an error if either starting
// or waiting fails, or if the context is canceled.
func (e *Executor) Run(
	ctx context.Context,
	request *executor.RunCommandRequest,
) (*models.RunCommandResult, error) {
	if err := e.Start(ctx, request); err != nil {
		return nil, err
	}
	resCh, errCh := e.Wait(ctx, request.ExecutionID)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case out := <-resCh:
		return out, nil
	case err := <-errCh:
		return nil, err
	}
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

// newDockerJobContainer is an internal method called by Start to set up a new Docker container
// for the job execution. It configures the container based on the provided dockerJobContainerParams.
// This includes decoding engine specifications, setting up environment variables, mounts, resource
// constraints, and network configurations. It then creates the container but does not start it.
// The method returns a container.CreateResponse and an error if any part of the setup fails.
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
	deviceRequests, deviceMappings, err := configureDevices(ctx, params.Resources)
	if err != nil {
		return container.CreateResponse{}, fmt.Errorf("creating container devices: %w", err)
	}
	log.Ctx(ctx).Trace().Msgf("Adding %d GPUs to request", params.Resources.GPU)

	hostConfig := &container.HostConfig{
		Mounts: mounts,
		Resources: container.Resources{
			Memory:         int64(params.Resources.Memory),
			NanoCPUs:       int64(params.Resources.CPU * NanoCPUCoefficient),
			DeviceRequests: deviceRequests,
			Devices:        deviceMappings,
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

func configureDevices(ctx context.Context, resources *models.Resources) ([]container.DeviceRequest, []container.DeviceMapping, error) {
	requests := []container.DeviceRequest{}
	mappings := []container.DeviceMapping{}
	vendorGroups := lo.GroupBy(resources.GPUs, func(gpu models.GPU) models.GPUVendor { return gpu.Vendor })

	for vendor, gpus := range vendorGroups {
		switch vendor {
		case models.GPUVendorNvidia:
			requests = append(requests, container.DeviceRequest{
				DeviceIDs:    lo.Map(gpus, func(gpu models.GPU, _ int) string { return fmt.Sprint(gpu.Index) }),
				Capabilities: [][]string{{"gpu"}},
			})
		case models.GPUVendorAMDATI:
			// https://docs.amd.com/en/latest/deploy/docker.html
			mappings = append(mappings, container.DeviceMapping{
				PathOnHost:        "/dev/kfd",
				PathInContainer:   "/dev/kfd",
				CgroupPermissions: "rwm",
			})
			fallthrough
		case models.GPUVendorIntel:
			// https://github.com/openvinotoolkit/docker_ci/blob/master/docs/accelerators.md
			paths := lo.FlatMap[models.GPU, string](gpus, func(gpu models.GPU, _ int) []string {
				return []string{
					filepath.Join("/dev/dri/by-path/", fmt.Sprintf("pci-%s-card", gpu.PCIAddress)),
					filepath.Join("/dev/dri/by-path/", fmt.Sprintf("pci-%s-render", gpu.PCIAddress)),
				}
			})

			for _, path := range paths {
				// We need to use the PCI address of the GPU to look up the correct devices to expose
				absPath, err := filepath.EvalSymlinks(path)
				if err != nil {
					return nil, nil, errors.Wrapf(err, "could not find attached device for GPU at %q", path)
				}

				mappings = append(mappings, container.DeviceMapping{
					PathOnHost:        absPath,
					PathInContainer:   absPath,
					CgroupPermissions: "rwm",
				})
			}
		default:
			return nil, nil, fmt.Errorf("job requires GPU from unsupported vendor %q", vendor)
		}
	}
	return requests, mappings, nil
}

func makeContainerMounts(
	ctx context.Context, inputs []storage.PreparedStorage, outputs []*models.ResultPath, resultsDir string) ([]mount.Mount, error) {
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

// FindRunningContainer, not part of the Executor interface, is a utility function that
// helps locate a container durin a restart check.
func (e *Executor) FindRunningContainer(ctx context.Context, executionID string) (string, error) {
	labelValue := labelExecutionValue(e.ID, executionID)
	return e.client.FindContainer(ctx, labelExecutionID, labelValue)
}

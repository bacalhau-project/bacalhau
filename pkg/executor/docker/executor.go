package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	dockermodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/resource"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/docker/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	pkgUtil "github.com/bacalhau-project/bacalhau/pkg/util"
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
	ID          string
	activeFlags map[string]chan struct{}
	client      *docker.Client
	cancellers  generic.SyncMap[string, context.CancelFunc]
}

func NewExecutor(
	_ context.Context,
	cm *system.CleanupManager,
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
	}

	cm.RegisterCallbackWithContext(de.cleanupAll)
	return de, nil
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
	log.Ctx(ctx).Info().Msgf("running execution %s", request.ExecutionID)
	ctx, cancel := context.WithCancel(ctx)
	e.cancellers.Put(request.ExecutionID, cancel)
	defer func() {
		if cancelFn, found := e.cancellers.Get(request.ExecutionID); found {
			e.cancellers.Delete(request.ExecutionID)
			cancelFn()
		}
	}()

	//nolint:ineffassign,staticcheck
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/executor/docker.Executor.Run")
	defer span.End()
	defer e.cleanupExecution(ctx, request.ExecutionID)

	dockerArgs, err := dockermodels.DecodeSpec(request.EngineParams)
	if err != nil {
		return nil, err
	}

	e.activeFlags[request.ExecutionID] = make(chan struct{}, 1)

	// the actual mounts we will give to the container
	// these are paths for both input and output data
	var mounts []mount.Mount
	for _, input := range request.Inputs {
		if input.Volume.Type == storage.StorageVolumeConnectorBind {
			log.Ctx(ctx).Trace().Msgf("Input Volume: %+v %+v", input.InputSource, input.Volume)

			mounts = append(mounts, mount.Mount{
				Type:     mount.TypeBind,
				ReadOnly: input.Volume.ReadOnly,
				Source:   input.Volume.Source,
				Target:   input.Volume.Target,
			})
		} else {
			return executor.FailResult(fmt.Errorf("unknown storage volume type: %s", input.Volume.Type))
		}
	}

	// for this phase of the outputs we ignore the engine because it's just about collecting the
	// data from the job and keeping it locally
	// the engine property of the output storage spec is how we will "publish" the output volume
	// if and when the deal is settled
	for _, output := range request.Outputs {
		if output.Name == "" {
			err = fmt.Errorf("output volume has no name: %+v", output)
			return executor.FailResult(err)
		}

		if output.Path == "" {
			err = fmt.Errorf("output volume has no Location: %+v", output)
			return executor.FailResult(err)
		}

		srcd := filepath.Join(request.ResultsDir, output.Name)
		err = os.Mkdir(srcd, util.OS_ALL_R|util.OS_ALL_X|util.OS_USER_W)
		if err != nil {
			return executor.FailResult(err)
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

	if _, set := os.LookupEnv("SKIP_IMAGE_PULL"); !set {
		dockerCreds := config.GetDockerCredentials()
		if pullErr := e.client.PullImage(ctx, dockerArgs.Image, dockerCreds); pullErr != nil {
			pullErr = errors.Wrapf(pullErr, docker.ImagePullError, dockerArgs.Image)
			return executor.FailResult(pullErr)
		}
	}

	containerConfig := &container.Config{
		Image:      dockerArgs.Image,
		Tty:        false,
		Env:        dockerArgs.EnvironmentVariables,
		Entrypoint: dockerArgs.Entrypoint,
		Cmd:        dockerArgs.Parameters,
		Labels:     e.containerLabels(request.ExecutionID, request.JobID),
		WorkingDir: dockerArgs.WorkingDirectory,
	}

	log.Ctx(ctx).Trace().Msgf("Container: %+v %+v", containerConfig, mounts)

	// Create GPU request if the job requests it
	var deviceRequests []container.DeviceRequest
	if request.Resources.GPU > 0 {
		deviceRequests = append(deviceRequests,
			container.DeviceRequest{
				DeviceIDs:    []string{"0"}, // TODO: how do we know which device ID to use?
				Capabilities: [][]string{{"gpu"}},
			},
		)
		log.Ctx(ctx).Trace().Msgf("Adding %d GPUs to request", request.Resources.GPU)
	}

	hostConfig := &container.HostConfig{
		Mounts: mounts,
		Resources: container.Resources{
			Memory:         int64(request.Resources.Memory),
			NanoCPUs:       int64(request.Resources.CPU * NanoCPUCoefficient),
			DeviceRequests: deviceRequests,
		},
	}

	// Create a network if the job requests it
	err = e.setupNetworkForJob(ctx, request.JobID, request.ExecutionID, request.Network, containerConfig, hostConfig)
	if err != nil {
		return executor.FailResult(err)
	}

	jobContainer, err := e.client.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		nil,
		nil,
		e.containerName(request.ExecutionID, request.JobID),
	)
	if err != nil {
		return executor.FailResult(errors.Wrap(err, "failed to create container"))
	}

	ctx = log.Ctx(ctx).With().Str("Container", jobContainer.ID).Logger().WithContext(ctx)

	e.activeFlags[request.ExecutionID] <- struct{}{}

	containerStartError := e.client.ContainerStart(
		ctx,
		jobContainer.ID,
		dockertypes.ContainerStartOptions{},
	)
	if containerStartError != nil {
		// Special error to alert people about bad executable
		internalContainerStartErrorMsg := "failed to start container"
		if strings.Contains(containerStartError.Error(), "executable file not found") {
			internalContainerStartErrorMsg = "Executable file not found"
		}
		internalContainerStartError := errors.Wrap(containerStartError, internalContainerStartErrorMsg)
		return executor.FailResult(internalContainerStartError)
	}

	// the idea here is even if the container errors
	// we want to capture stdout, stderr and feed it back to the user
	var containerError error
	var containerExitStatusCode int64
	statusCh, errCh := e.client.ContainerWait(
		ctx,
		jobContainer.ID,
		container.WaitConditionNotRunning,
	)
	select {
	case err = <-errCh:
		containerError = err
	case exitStatus := <-statusCh:
		containerExitStatusCode = exitStatus.StatusCode
		if exitStatus.Error != nil {
			containerError = errors.New(exitStatus.Error.Message)
		}
	}

	// Can't use the original context as it may have already been timed out
	detachedContext, cancel := context.WithTimeout(pkgUtil.NewDetachedContext(ctx), 3*time.Second)
	defer cancel()
	stdoutPipe, stderrPipe, logsErr := e.client.FollowLogs(detachedContext, jobContainer.ID)

	log.Ctx(detachedContext).Debug().Err(logsErr).Msg("Captured stdout/stderr for container")

	return executor.WriteJobResults(
		request.ResultsDir,
		stdoutPipe,
		stderrPipe,
		int(containerExitStatusCode),
		multierr.Combine(containerError, logsErr),
		request.OutputLimits,
	)
}

func (e *Executor) Cancel(ctx context.Context, id string) error {
	log.Ctx(ctx).Trace().Msgf("canceling execution %s", id)
	if cancel, found := e.cancellers.Get(id); found {
		e.cancellers.Delete(id)
		cancel()
	}
	log.Ctx(ctx).Debug().Msgf("canceled execution %s", id)

	return nil
}

func (e *Executor) GetOutputStream(ctx context.Context, executionID string, withHistory bool, follow bool) (io.ReadCloser, error) {
	// We have to wait until the condition is met otherwise we may be here too early and
	// the container isn't created yet. The channel in the activeFlags map will either have
	// a value waiting, or have one written to it shortly
	c, present := e.activeFlags[executionID]
	if present {
		log.Ctx(ctx).Debug().Msg("waiting for container to be created to read logs")
		<-c

		// Delete the channel from the map, any further followers will skip this section
		// as the absence of the channel suggests the container is already created.
		delete(e.activeFlags, executionID)
	}

	ctrID, err := e.client.FindContainer(ctx, labelExecutionID, e.labelExecutionValue(executionID))
	if err != nil {
		return nil, err
	}

	since := strconv.FormatInt(time.Now().Unix(), 10) //nolint:gomnd
	if withHistory {
		since = "1"
	}

	// Gets the underlying reader, and provides data since the value of the `since` timestamp.
	// If we want everything, we specify 1, a timestamp which we are confident we don't have
	// logs before. If we want to just follow new logs, we pass `time.Now()` as a string.
	reader, err := e.client.GetOutputStream(ctx, ctrID, since, follow)
	if err != nil {
		return nil, err
	}

	return reader, nil
}

func (e *Executor) cleanupExecution(ctx context.Context, executionID string) {
	// Use a detached context in case the current one has already been canceled
	separateCtx, cancel := context.WithTimeout(pkgUtil.NewDetachedContext(ctx), 1*time.Minute)
	defer cancel()
	if config.ShouldKeepStack() || !e.client.IsInstalled(separateCtx) {
		return
	}

	// Attempt to delete the channel that was used to mark that the container exists
	c, present := e.activeFlags[executionID]
	if present {
		close(c)
		delete(e.activeFlags, executionID)
	}

	err := e.client.RemoveObjectsWithLabel(separateCtx, labelExecutionID, e.labelExecutionValue(executionID))
	logLevel := map[bool]zerolog.Level{true: zerolog.DebugLevel, false: zerolog.ErrorLevel}[err == nil]
	log.Ctx(ctx).WithLevel(logLevel).Err(err).Msg("Cleaned up job Docker resources")
}

func (e *Executor) cleanupAll(ctx context.Context) error {
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

func (e *Executor) dockerObjectName(executionID string, jobID string, parts ...string) string {
	strs := []string{"bacalhau", e.ID, jobID, executionID}
	strs = append(strs, parts...)
	return strings.Join(strs, "-")
}

func (e *Executor) containerName(executionID string, jobID string) string {
	return e.dockerObjectName(executionID, jobID, "executor")
}

func (e *Executor) containerLabels(executionID string, jobID string) map[string]string {
	return map[string]string{
		labelExecutorName: e.ID,
		labelJobName:      e.labelJobValue(jobID),
		labelExecutionID:  e.labelExecutionValue(executionID),
	}
}

func (e *Executor) labelJobValue(jobID string) string {
	return e.ID + jobID
}

func (e *Executor) labelExecutionValue(executionID string) string {
	return e.ID + executionID
}

// Compile-time interface check:
var _ executor.Executor = (*Executor)(nil)

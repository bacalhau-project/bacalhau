package docker

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.uber.org/atomic"
	"go.uber.org/multierr"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	pkgUtil "github.com/bacalhau-project/bacalhau/pkg/util"
)

type executionHandler struct {
	// provided by the executor
	client *docker.Client
	logger zerolog.Logger

	// meta data about the task
	executionID string
	containerID string
	resultsDir  string
	limits      executor.OutputLimits

	// synchronization
	// blocks until the container starts
	activeCh chan bool
	// blocks until the run method returns
	waitCh chan bool
	// true until the run method returns
	running *atomic.Bool

	// results
	result *handlerResult
}

type handlerResult struct {
	err      error
	exitcode int64
	stdOut   io.Reader
	stdErr   io.Reader
}

func (h *executionHandler) run(ctx context.Context) {
	h.running.Store(true)
	defer func() {
		h.running.Store(false)
		close(h.waitCh)
		// the context is getting canceled by something before this method can complete...
		if err := h.destroy(pkgUtil.NewDetachedContext(ctx)); err != nil {
			log.Warn().Err(err).Msg("failed to cleanup container")
		}
	}()
	// start the container
	h.logger.Info().Msg("starting container execution")
	if err := h.client.ContainerStart(ctx, h.containerID, dockertypes.ContainerStartOptions{}); err != nil {
		// Special error to alert people about bad executable
		internalContainerStartErrorMsg := "failed to start container"
		if strings.Contains(err.Error(), "executable file not found") {
			internalContainerStartErrorMsg = "executable file not found"
		}
		startError := errors.Wrap(err, internalContainerStartErrorMsg)

		h.logger.Warn().Err(startError).Msg("failed to start container")
		h.result.err = fmt.Errorf("failed to start container: %w", startError)
		// we failed to start the container, bail.
		return
	}
	// The container is now active
	close(h.activeCh)

	// the idea here is even if the container errors
	// we want to capture stdout, stderr and feed it back to the user
	var containerError error
	var containerExitStatusCode int64
	statusCh, errCh := h.client.ContainerWait(ctx, h.containerID, container.WaitConditionNotRunning)
	select {
	/*
		case <-ctx.Done():
			h.logger.Err(ctx.Err()).Msg("context canceled while waiting for container status")

	*/
	// TODO ContainerWait will either return an error on errCh OR a status that contains an error
	// should we bail if the former is true?
	case err := <-errCh:
		h.logger.Warn().Err(containerError).Msg("failed to wait on container")
		containerError = err
	case exitStatus := <-statusCh:
		containerExitStatusCode = exitStatus.StatusCode
		if exitStatus.Error != nil {
			h.logger.Warn().
				Str("error", exitStatus.Error.Message).
				Int64("code", exitStatus.StatusCode).
				Msg("container returned status with error")
			containerError = errors.New(exitStatus.Error.Message)
		} else {
			h.logger.Info().
				Int64("code", exitStatus.StatusCode).
				Msg("received status from container")
		}
	}

	// Can't use the original context as it may have already been timed out
	// TODO I am dubious of this timeout, is it necessary? 3 seconds?
	detachedContext, cancel := context.WithTimeout(pkgUtil.NewDetachedContext(ctx), 3*time.Second)
	defer cancel()
	// TODO there is a race condition here, we may not have followed all logs before the method returns
	// I suspect we may need to perform this operation in sync before exiting as its returned readers may not have
	// any content
	stdoutPipe, stderrPipe, err := h.client.FollowLogs(detachedContext, h.containerID)
	if err != nil {
		h.logger.Warn().Err(err).Msg("failed to follow container logs")
		h.result.err = multierr.Combine(containerError, err)
		return
	}
	h.logger.Info().Msg("captured stdout/stderr for container")

	h.result = &handlerResult{
		err:      containerError,
		exitcode: containerExitStatusCode,
		stdOut:   stdoutPipe,
		stdErr:   stderrPipe,
	}

	h.logger.Info().
		Int64("status", containerExitStatusCode).
		Msg("container execution ended")
	return
}

func (h *executionHandler) kill(ctx context.Context) error {
	// TODO pass a signal, which we can do by modifying this client wrapper to accept one.
	// the wrapped docker client supports such params.
	h.logger.Info().Msg("killing the container")
	// NB(forrest): stopping the container here will cause the run method to perform cleanup if still active, else noop.
	return h.client.ContainerStop(ctx, h.containerID, time.Second)
}

func (h *executionHandler) destroy(ctx context.Context) error {
	h.logger.Info().Msg("destroying the container")
	errs := make([]error, 0, 4)
	if err := h.kill(ctx); err != nil {
		h.logger.Warn().Err(err).Msg("failed to kill container")
		errs = append(errs, err)
	}
	info, err := h.client.ContainerInspect(ctx, h.containerID)
	if err != nil {
		h.logger.Warn().Err(err).Msg("failed to inspect container")
		errs = append(errs, err)
	}
	h.logger.Info().Msg("removing container")
	if err := h.client.RemoveContainer(ctx, h.containerID); err != nil {
		h.logger.Warn().Err(err).Msg("failed to remove container")
		errs = append(errs, err)
	}
	for networkID := range info.NetworkSettings.Networks {
		h.logger.Info().Str("networkID", networkID).Msg("removing container network")
		if err := h.client.NetworkRemove(ctx, networkID); err != nil {
			h.logger.Warn().Err(err).Msg("failed to remove network")
			errs = append(errs, err)
		}
	}
	return multierr.Combine(errs...)
}

func (h *executionHandler) outputStream(ctx context.Context, withHistory, follow bool) (io.ReadCloser, error) {
	since := strconv.FormatInt(time.Now().Unix(), 10) //nolint:gomnd
	if withHistory {
		since = "1"
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	// We have to wait until the condition is met otherwise we may be here too early and
	// the container isn't created yet.
	case <-h.activeCh:
	}
	// Gets the underlying reader, and provides data since the value of the `since` timestamp.
	// If we want everything, we specify 1, a timestamp which we are confident we don't have
	// logs before. If we want to just follow new logs, we pass `time.Now()` as a string.
	return h.client.GetOutputStream(ctx, h.containerID, since, follow)
}

func (h *executionHandler) active() bool {
	return h.running.Load()
}

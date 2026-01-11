package docker

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/atomic"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
)

type executionHandler struct {
	//
	// provided by the executor
	client *docker.Client
	logger zerolog.Logger
	// meta data about the executor
	ID string

	//
	// meta data about the task
	executionID  string
	containerID  string
	executionDir string
	limits       executor.OutputLimits
	keepStack    bool

	//
	// synchronization
	// blocks until the container starts
	activeCh chan bool
	// blocks until the run method returns
	waitCh chan bool
	// true until the run method returns
	running *atomic.Bool
	// cancel function
	cancelFunc context.CancelCauseFunc

	//
	// results
	result *models.RunCommandResult
}

//nolint:funlen
func (h *executionHandler) run(ctx context.Context) {
	ActiveExecutions.Inc(ctx, attribute.String("executor_id", h.ID))
	h.running.Store(true)
	defer func() {
		destroyTimeout := time.Second * 10
		if err := h.destroy(destroyTimeout); err != nil {
			log.Warn().Err(err).Msg("failed to cleanup container")
		}
		h.running.Store(false)
		close(h.waitCh)
		ActiveExecutions.Dec(ctx, attribute.String("executor_id", h.ID))
	}()

	// start the container
	h.logger.Info().Msg("starting container execution")

	if err := h.client.ContainerStart(ctx, h.containerID, container.StartOptions{}); err != nil {
		// Special error to alert people about bad executable
		internalContainerStartErrorMsg := "failed to start container"
		if strings.Contains(err.Error(), "executable file not found") {
			internalContainerStartErrorMsg = "executable file not found"
		}
		startError := errors.Wrap(err, internalContainerStartErrorMsg)

		h.logger.Warn().Err(startError).Msg("failed to start container")
		h.result = executor.NewFailedResult(fmt.Sprintf("failed to start container: %s", startError))
		// we failed to start the container, bail.
		return
	}

	// The container is now active
	close(h.activeCh)

	// Capture the container logs in a separate goroutine.
	logCaptureErrCh := make(chan error, 1)
	go func() {
		logCaptureErrCh <- h.captureContainerLogs(ctx)
	}()

	// the idea here is even if the container errors
	// we want to capture stdout, stderr and feed it back to the user
	var containerError error
	var containerExitStatusCode int64
	statusCh, errCh := h.client.ContainerWait(ctx, h.containerID, container.WaitConditionNotRunning)
	select {
	case <-ctx.Done():
		// failure case, the context has been canceled. We are aborting this execution
		cause := context.Cause(ctx)
		if cause == nil {
			cause = fmt.Errorf("context canceled while waiting on container status: %w", ctx.Err())
		}
		h.logger.Err(cause).Msg("cancel waiting on container status")
		h.result = executor.NewFailedResult(cause.Error())
		// the context was canceled, bail.
		return
	case err := <-errCh:
		// the docker client failed to begin the wait request or failed to get a response. We are aborting this execution.
		reason := fmt.Errorf("received error response from docker client while waiting on container: %w", err)
		h.logger.Warn().Err(reason).Msg("failed while waiting on container status")
		h.result = executor.NewFailedResult(reason.Error())
		// the docker client was unable to wait on the container, bail.
		return
	case exitStatus := <-statusCh:
		// success case, the container completed its execution, but may have experienced an error, we will attempt to collect logs.
		containerExitStatusCode = exitStatus.StatusCode
		containerJSON, err := h.client.ContainerInspect(ctx, h.containerID)
		if err != nil {
			h.logger.Warn().Err(err).Msg("failed to inspect docker container")
			h.result = &models.RunCommandResult{
				ExitCode: int(containerExitStatusCode),
				ErrorMsg: err.Error(),
			}
			return
		}
		if containerJSON.State.OOMKilled {
			containerError = errors.New(`memory limit exceeded`) //nolint:lll
			h.result = &models.RunCommandResult{
				ExitCode: int(containerExitStatusCode),
				ErrorMsg: containerError.Error(),
			}
			return
		}
		if exitStatus.Error != nil {
			h.logger.Warn().
				Str("error", exitStatus.Error.Message).
				Int64("status", exitStatus.StatusCode).
				Msg("container returned status with error")
			containerError = errors.New(exitStatus.Error.Message)
		} else {
			h.logger.Info().
				Int64("status", exitStatus.StatusCode).
				Msg("received status from container")
		}
	}

	// Wait for the log capture to finish and populate the Run result
	logCaptureErr := <-logCaptureErrCh
	if logCaptureErr != nil {
		h.result = executor.NewFailedResult(fmt.Sprintf("failed to capture container output: %s", logCaptureErr))
		return
	}

	// TODO: We no longer need to ask Docker for the logs again as we already have them written to a file.
	stdoutPipe, stderrPipe, err := h.client.FollowLogs(ctx, h.containerID)
	if err != nil {
		h.logger.Warn().Err(err).Msg("failed to follow container logs")

		followError := fmt.Errorf("failed to follow container logs: %w", err)
		// if there was also a container error record it.
		if containerError != nil {
			h.result = &models.RunCommandResult{
				ExitCode: int(containerExitStatusCode),
				ErrorMsg: fmt.Sprintf("container error: '%s'. logs error: '%s'", containerError, followError),
			}
		} else {
			h.result = &models.RunCommandResult{
				ExitCode: int(containerExitStatusCode),
				ErrorMsg: followError.Error(),
			}
		}
		// we don't have a reader for stderr/out, so just return the error and exit code.
		return
	}

	h.logger.Info().Msg("captured stdout/stderr for container")
	// we successfully followed the container logs, the container may still have produced and error which we will record
	// along with a truncated version of the logs.
	// persist stderr/out to the results directory, and store the metadata in the handler.
	resultsDir := compute.ExecutionResultsDir(h.executionDir)
	h.result = executor.WriteJobResults(resultsDir, stdoutPipe, stderrPipe, int(containerExitStatusCode), containerError, h.limits)

	h.logger.Info().
		Int64("status", containerExitStatusCode).
		Msg("container execution ended")
}

func (h *executionHandler) kill(ctx context.Context) error {
	// TODO pass a signal, which we can do by modifying this client wrapper to accept one.
	// the wrapped docker client supports such params.
	h.logger.Info().Msg("killing the container")
	// NB(forrest): stopping the container here will cause the run method to perform cleanup if still active, else noop.
	return h.client.ContainerStop(ctx, h.containerID, time.Second)
}

func (h *executionHandler) destroy(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	h.logger.Info().Msg("destroying the container")

	// stop the container
	if err := h.kill(ctx); err != nil {
		return fmt.Errorf("failed to kill container (%s): %w", h.containerID, err)
	}

	// TODO document why this configuration value exists.
	if !h.keepStack {
		h.logger.Info().Msg("removing container")
		if err := h.client.RemoveContainer(ctx, h.containerID); err != nil {
			return err
		}
		return h.client.RemoveObjectsWithLabel(ctx, labelExecutionID, labelExecutionValue(h.ID, h.executionID))
	}
	return nil
}

// TODO: Log streaming should be moved outside of executor/handler and just rely on local files captured during container execution.
func (h *executionHandler) outputStream(ctx context.Context, request messages.ExecutionLogsRequest) (io.ReadCloser, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	// We have to wait until the condition is met otherwise we may be here too early and
	// the container isn't created yet.
	case <-h.activeCh:
	}

	// Read and filter container logs from the local file
	logsDir := compute.ExecutionLogsDir(h.executionDir)
	reader, err := logstream.NewReaderForRequest(logsDir, request)
	if err != nil {
		return nil, err
	}

	return reader, nil
}

func (h *executionHandler) active() bool {
	return h.running.Load()
}

func (h *executionHandler) captureContainerLogs(ctx context.Context) error {
	// Get a reader from the container
	logStreamReader, err := h.client.GetOutputStream(ctx, h.containerID, nil, true, true)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to capture container logs")
		return err
	}
	defer closer.CloseWithLogOnError("container_logs_reader", logStreamReader)

	logsDir := compute.ExecutionLogsDir(h.executionDir)

	// Create a writer to write the logs to a file
	logWriter, err := logstream.NewExecutionLogWriter(logsDir)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to create container logs file writer")
		return err
	}
	defer closer.CloseWithLogOnError("container_logs_file_writer", logWriter)

	// Copy container logs to the file, adding an end frame when the reader returns EOF
	written, err := logstream.StdCopyWithEndFrame(logWriter, logStreamReader)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to capture container logs")
		return err
	}

	h.logger.Debug().
		Int64("byte_size", written).
		Err(err).
		Str("logs_dir", logsDir).
		Msg("container logs captured")
	return nil
}

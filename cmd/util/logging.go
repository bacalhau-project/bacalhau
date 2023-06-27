package util

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
)

var LoggingMode = logger.LogModeDefault

const (
	JSONFormat string = "json"
	YAMLFormat string = "yaml"
)

func Logs(cmd *cobra.Command, jobID string, follow, history bool) error {
	ctx := cmd.Context()

	requestedJobID := jobID
	if requestedJobID == "" {
		var byteResult []byte
		byteResult, err := ReadFromStdinIfAvailable(cmd)
		if err != nil {
			return fmt.Errorf("unknown error reading from file: %w", err)
		}
		requestedJobID = string(byteResult)
	}

	// After retrieving the job to ensure it exists, we want to find an execution
	// that is currently in an active state to read the logs from. In future we
	// may be smarter about handling multiple executions, but in the short term
	// this will handle the most common case of wanting output from a single running
	// job.
	apiClient := GetAPIClient(ctx)
	job, jobFound, err := apiClient.Get(ctx, requestedJobID)
	if err != nil {
		return err
	}

	if !jobFound {
		return fmt.Errorf("could not find job %s", requestedJobID)
	}

	jobID = job.Job.ID()
	executionID := ""
	for _, execution := range job.State.Executions {
		if execution.State.IsActive() {
			executionID = execution.ComputeReference
		}
	}

	if executionID == "" {
		return fmt.Errorf("unable to find an active execution for job (ID: %s)", jobID)
	}

	// Get a websocket connection to the requester node from where we will be streamed
	// any dataframes that are logged from the requested execution/job.
	conn, err := apiClient.Logs(ctx, jobID, executionID, history, follow)
	if err != nil {
		if errResp, ok := err.(*bacerrors.ErrorResponse); ok {
			return errResp
		} else {
			return fmt.Errorf("unknown error trying to stream logs from job (ID: %s): %w", requestedJobID, errResp)
		}
	}
	defer conn.Close()

	if err := readLogoutput(ctx, cmd, conn); err != nil {
		return fmt.Errorf("reading log output: %w", err)
	}
	return nil
}

// ApplyPorcelainLogLevel sets the log level of loggers running on user-facing
// "porcelain" commands to be zerolog.FatalLevel to reduce noise shown to users.
func ApplyPorcelainLogLevel(cmd *cobra.Command, _ []string) {
	if _, err := zerolog.ParseLevel(os.Getenv("LOG_LEVEL")); err != nil {
		return
	}

	ctx := cmd.Context()
	ctx = log.Ctx(ctx).Level(zerolog.FatalLevel).WithContext(ctx)
	cmd.SetContext(ctx)
}

type Msg struct {
	Tag  uint8
	Data string
}

func readLogoutput(ctx context.Context, cmd *cobra.Command, conn *websocket.Conn) error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	exiting := false
	done := make(chan struct{})

	go func() {
		var fd *os.File
		var msg Msg

		defer close(done)
		for !exiting {
			err := conn.ReadJSON(&msg)
			if err != nil {
				// If the error is NOT a CloseNormal then log the error
				if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
					cmd.PrintErrf("failed to read: %s", err)
				}

				exiting = true
			}

			if msg.Tag == 1 {
				fd = os.Stdout
			} else if msg.Tag == 2 {
				fd = os.Stderr
			}
			n, err := fd.WriteString(msg.Data)
			if err != nil {
				if !exiting {
					cmd.PrintErrf("failed to write: %s", err)
				}
				break
			}
			if n != len(msg.Data) {
				cmd.PrintErrf(
					"failed to write to fd, tried to write %d bytes but only managed %d",
					len(msg.Data),
					n,
				)
			}
		}
	}()

	for {
		select {
		case <-done:
			exiting = true
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return nil
		case <-ctx.Done():
			exiting = true
			return nil
		case <-interrupt:
			exiting = true
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

			select {
			case <-done:
			case <-time.After(time.Second):
			}

			return nil
		}
	}

	// unreachable
}

package util

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
)

var LoggingMode = logger.LogModeDefault

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
		}
		return fmt.Errorf("unknown error trying to stream logs from job (ID: %s): %w", requestedJobID, err)
	}
	defer conn.Close()

	if err := readLogoutput(ctx, cmd, conn); err != nil {
		return fmt.Errorf("reading log output: %w", err)
	}
	return nil
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
		var msg models.ExecutionLog

		defer close(done)
		for !exiting {
			err := conn.ReadJSON(&msg)
			if err != nil {
				// If the error is NOT a CloseNormal then log the error
				if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
					cmd.PrintErrf("Error: failed to read message: %s", err)
				}

				exiting = true
				continue
			}

			if msg.Error != "" {
				var errResponse bacerrors.ErrorResponse
				err := json.Unmarshal([]byte(msg.Error), &errResponse)
				if err != nil {
					Fatal(cmd, fmt.Errorf("failed decoding error message from server: %s", err), 1)
				}

				e := fmt.Sprintf("Error: %s", &errResponse)
				Fatal(cmd, errors.New(e), 1)

				_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				conn.Close()

				return
			}

			fd = os.Stdout
			n, err := fd.WriteString(msg.Line)
			if err != nil {
				if !exiting {
					cmd.PrintErrf("failed to write: %s", err)
				}
				break
			}
			if n != len(msg.Line) {
				cmd.PrintErrf(
					"failed to write to fd, tried to write %d bytes but only managed %d",
					len(msg.Line),
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

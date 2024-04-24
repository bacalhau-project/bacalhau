package util

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
)

var LoggingMode = logger.LogModeDefault

type LogOptions struct {
	JobID       string
	ExecutionID string
	Follow      bool
	Tail        bool
}

func Logs(cmd *cobra.Command, options LogOptions) error {
	requestedJobID := options.JobID
	if requestedJobID == "" {
		var byteResult []byte
		byteResult, err := ReadFromStdinIfAvailable(cmd)
		if err != nil {
			return fmt.Errorf("unknown error reading from file: %w", err)
		}
		requestedJobID = string(byteResult)
	}

	// TODO(forrest) [fixme]
	apiClient := GetAPIClientV2(cmd, nil, nil)
	ch, err := apiClient.Jobs().Logs(cmd.Context(), &apimodels.GetLogsRequest{
		JobID:       options.JobID,
		ExecutionID: options.ExecutionID,
		Follow:      options.Follow,
		Tail:        options.Tail,
	})
	if err != nil {
		if errResp, ok := err.(*bacerrors.ErrorResponse); ok {
			return errResp
		}
		return fmt.Errorf("unknown error trying to stream logs from job (ID: %s): %w", requestedJobID, err)
	}

	if err := readLogoutput(cmd.Context(), ch); err != nil {
		return fmt.Errorf("error reading log output: %w", err)
	}
	return nil
}

func readLogoutput(ctx context.Context, logsChannel <-chan *concurrency.AsyncResult[models.ExecutionLog]) error {
	fd := os.Stdout
	for {
		select {
		case result, ok := <-logsChannel:
			if !ok {
				return nil
			}
			if result.Err != nil {
				return fmt.Errorf("error received from server: %w", result.Err)
			}

			msg := result.Value
			n, err := fd.WriteString(msg.Line)
			if err != nil {
				return fmt.Errorf("failed to write to fd: %w", err)
			}
			if n != len(msg.Line) {
				return fmt.Errorf("failed to write to fd, tried to write %d bytes but only managed %d", len(msg.Line), n)
			}
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.Canceled) {
				return nil
			}
			return ctx.Err()
		}
	}
	// unreachable
}

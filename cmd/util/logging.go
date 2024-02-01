package util

import (
	"context"
	"fmt"
	"os"

	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
)

var LoggingMode = logger.LogModeDefault

func Logs(cmd *cobra.Command, jobID string, follow, history bool) error {
	requestedJobID := jobID
	if requestedJobID == "" {
		var byteResult []byte
		byteResult, err := ReadFromStdinIfAvailable(cmd)
		if err != nil {
			return fmt.Errorf("unknown error reading from file: %w", err)
		}
		requestedJobID = string(byteResult)
	}

	apiClient := GetAPIClientV2()
	ch, err := apiClient.Jobs().Logs(cmd.Context(), &apimodels.GetLogsRequest{
		JobID:       requestedJobID,
		Follow:      follow,
		WithHistory: history,
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
			return nil
		}
	}
	// unreachable
}

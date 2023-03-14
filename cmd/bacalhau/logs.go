package bacalhau

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
)

var (
	logsShortDesc = templates.LongDesc(i18n.T(`
		Follow logs from a currently executing job
`))

	//nolint:lll // Documentation
	logsExample = templates.Examples(i18n.T(`
		# Follow logs for a previously submitted job
		bacalhau logs 51225160-807e-48b8-88c9-28311c7899e1

		# Follow output with a short ID 
		bacalhau logs ebd9bf2f
`))
)

type Msg struct {
	Tag  uint8
	Data string
}

type LogCommandOptions struct {
	WithHistory bool
}

func newLogsCmd() *cobra.Command {
	options := LogCommandOptions{}

	logsCmd := &cobra.Command{
		Use:     "logs [id]",
		Short:   logsShortDesc,
		Example: logsExample,
		Args:    cobra.ExactArgs(1),
		PreRun:  applyPorcelainLogLevel,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			return logs(cmd, cmdArgs, options)
		},
	}

	logsCmd.PersistentFlags().BoolVarP(
		&options.WithHistory, "all", "a", false,
		`Show the entire log history`,
	)

	return logsCmd
}

func logs(cmd *cobra.Command, cmdArgs []string, options LogCommandOptions) error {
	ctx := cmd.Context()

	requestedJobID := cmdArgs[0]
	if requestedJobID == "" {
		var byteResult []byte
		byteResult, err := ReadFromStdinIfAvailable(cmd, cmdArgs)
		if err != nil {
			Fatal(cmd, fmt.Sprintf("Unknown error reading from file: %s\n", err), 1)
			return err
		}
		requestedJobID = string(byteResult)
	}

	// After retrieving the job to ensure it exists, we want to find an execution
	// that is currently in an active state to read the logs from. In future we
	// may be smarter about handling multiple executions, but in the short term
	// this will handle the most common case of wanting output from a single running
	// job.
	apiClient := GetAPIClient()
	job, jobFound, err := apiClient.Get(ctx, requestedJobID)
	if err != nil {
		Fatal(cmd, err.Error(), 1)
		return nil
	}

	if !jobFound {
		Fatal(cmd, fmt.Sprintf("could not find job %s", requestedJobID), 1)
	}

	jobID := job.Job.ID()
	executionID := ""
	for _, execution := range job.State.Executions {
		if execution.State.IsActive() {
			executionID = execution.ComputeReference
		}
	}

	if executionID == "" {
		Fatal(cmd, fmt.Sprintf("Unable to find an active execution for job (ID: %s)", jobID), 1)
	}

	// Get a websocket connection to the requester node from where we will be streamed
	// any dataframes that are logged from the requested execution/job.
	conn, err := apiClient.Logs(ctx, jobID, executionID, options.WithHistory)
	if err != nil {
		if er, ok := err.(*bacerrors.ErrorResponse); ok {
			Fatal(cmd, er.Error(), 1)
			return nil
		} else {
			Fatal(cmd, fmt.Sprintf("Unknown error trying to stream logs from job (ID: %s): %+v", requestedJobID, err), 1)
			return nil
		}
	}
	defer conn.Close()

	return readLogoutput(ctx, cmd, conn)
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
				if !exiting {
					cmd.PrintErrf("failed to read: %s", err)
				}
				break
			}

			if msg.Tag == 1 {
				fd = os.Stdout
			} else if msg.Tag == 2 {
				fd = os.Stderr
			}
			n, err := fd.WriteString(msg.Data)
			if err != nil {
				cmd.PrintErrf("failed to write: %s", err)
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
			return conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		case <-ctx.Done():
			exiting = true
			return nil
		case <-interrupt:
			exiting = true
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				return err
			}

			select {
			case <-done:
			case <-time.After(time.Second):
			}

			return nil
		}
	}

	// unreachable
}

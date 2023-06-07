package cancel

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/cmd/bacalhau/handler"
	printer "github.com/bacalhau-project/bacalhau/cmd/printer"
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
)

var (
	cancelLong = templates.LongDesc(i18n.T(`
		Cancel a previously submitted job.
`))

	//nolint:lll // Documentation
	cancelExample = templates.Examples(i18n.T(`
		# Cancel a previously submitted job
		bacalhau cancel 51225160-807e-48b8-88c9-28311c7899e1

		# Cancel a job, with a short ID.
		bacalhau cancel ebd9bf2f
`))
)

var (
	checkingJobStatusMessage = i18n.T("Checking job status")
	connectingMessage        = i18n.T("Connecting to network")
	gettingJobMessage        = i18n.T("Verifying job state")
	cancellingJobMessage     = i18n.T("Canceling job")

	jobAlreadyCompleteMessage = i18n.T(`Job is already in a terminal state.
The current state is: %s
`)
)

type CancelOptions struct {
	Quiet bool
}

func NewCancelOptions() *CancelOptions {
	return &CancelOptions{
		Quiet: false,
	}
}

func NewCmd() *cobra.Command {
	cancelOptions := NewCancelOptions()

	cancelCmd := &cobra.Command{
		Use:     "cancel [id]",
		Short:   "Cancel a previously submitted job",
		Long:    cancelLong,
		Example: cancelExample,
		Args:    cobra.ExactArgs(1),
		PreRun:  handler.ApplyPorcelainLogLevel,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			err, exitcode := cancel(cmd, cmdArgs, cancelOptions)
			if err != nil {
				handler.Fatal(cmd, err, exitcode)
			}
			return nil
		},
	}

	cancelCmd.PersistentFlags().BoolVar(
		&cancelOptions.Quiet, "quiet", cancelOptions.Quiet,
		`Do not print anything to stdout or stderr`,
	)
	return cancelCmd
}

func cancel(cmd *cobra.Command, cmdArgs []string, options *CancelOptions) (error, int) {
	ctx := cmd.Context()

	if options.Quiet {
		cmd.SetOutput(io.Discard)
	}

	cmd.Printf("%s\n\n", checkingJobStatusMessage)

	widestString := findWidestString(
		checkingJobStatusMessage,
		connectingMessage,
		gettingJobMessage,
		cancellingJobMessage,
	)

	writer := cmd.OutOrStdout()
	if options.Quiet {
		writer = io.Discard
	}
	// Create a spinner that will exit if/when it sees ctrl-c
	spinner, err := printer.NewSpinner(ctx, writer, widestString, true)
	if err != nil {
		return err, handler.ExitError
	}
	spinner.Run()

	requestedJobID := cmdArgs[0]
	if requestedJobID == "" {
		var byteResult []byte
		byteResult, err = handler.ReadFromStdinIfAvailable(cmd)
		if err != nil {
			return fmt.Errorf("unknown error reading from file: %s", err), handler.ExitError
		}
		requestedJobID = string(byteResult)
	}

	// Let the user know we are initiating the request
	spinner.NextStep(connectingMessage)
	apiClient := handler.GetAPIClient(ctx)

	// Fetch the job information so we can check whether the task is already
	// terminal or not. We will not send requests if it is.
	spinner.NextStep(gettingJobMessage)
	job, jobFound, err := apiClient.Get(ctx, requestedJobID)
	if err != nil {
		spinner.Done(printer.StopFailed)
		return err, handler.ExitError
	}

	if !jobFound {
		spinner.Done(printer.StopFailed)
	}

	// Check status to make sure there is something to be canceled. If it is currently
	// in a terminal state, then we'll exit immediately
	if job.State.State.IsTerminal() {
		spinner.Done(printer.StopFailed)
		errorMessage := fmt.Errorf(jobAlreadyCompleteMessage, job.State.State.String())
		return errorMessage, handler.ExitError
	}

	// Submit a request to cancel the specified job. It is the responsibility of the
	// requester to decide if we are allowed to do that or not.
	spinner.NextStep(cancellingJobMessage)

	jobState, err := apiClient.Cancel(ctx, job.Job.Metadata.ID, "Canceled at user request")
	if err != nil {
		spinner.Done(printer.StopFailed)

		if err, ok := err.(*bacerrors.ErrorResponse); ok {
			return err, handler.ExitError
		} else {
			return fmt.Errorf("unknown error trying to cancel job (ID: %s): %+v", requestedJobID, err), handler.ExitError
		}
	}

	spinner.Done(printer.StopSuccess)
	cmd.Printf("\nJob successfully canceled. Job ID: %s\n", jobState.JobID)

	return nil, handler.ExitSuccess
}

func findWidestString(messages ...string) int {
	widest := 0
	for _, msg := range messages {
		msgLen := len(msg)
		if msgLen > widest {
			widest = msgLen
		}
	}
	return widest
}

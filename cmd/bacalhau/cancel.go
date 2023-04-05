package bacalhau

import (
	"fmt"
	"io"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
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

func newCancelCmd() *cobra.Command {
	cancelOptions := NewCancelOptions()

	cancelCmd := &cobra.Command{
		Use:     "cancel [id]",
		Short:   "Cancel a previously submitted job",
		Long:    cancelLong,
		Example: cancelExample,
		Args:    cobra.ExactArgs(1),
		PreRun:  applyPorcelainLogLevel,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			return cancel(cmd, cmdArgs, cancelOptions)
		},
	}

	cancelCmd.PersistentFlags().BoolVar(
		&cancelOptions.Quiet, "quiet", cancelOptions.Quiet,
		`Do not print anything to stdout or stderr`,
	)
	return cancelCmd
}

func cancel(cmd *cobra.Command, cmdArgs []string, options *CancelOptions) error {
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
	spinner, err := NewSpinner(ctx, writer, widestString, true)
	if err != nil {
		Fatal(cmd, err.Error(), 1)
	}
	spinner.Run()

	requestedJobID := cmdArgs[0]
	if requestedJobID == "" {
		var byteResult []byte
		byteResult, err = ReadFromStdinIfAvailable(cmd, cmdArgs)
		if err != nil {
			Fatal(cmd, fmt.Sprintf("Unknown error reading from file: %s\n", err), 1)
			return err
		}
		requestedJobID = string(byteResult)
	}

	// Let the user know we are initiating the request
	spinner.NextStep(connectingMessage)
	apiClient := GetAPIClient()

	// Fetch the job information so we can check whether the task is already
	// terminal or not. We will not send requests if it is.
	spinner.NextStep(gettingJobMessage)
	job, jobFound, err := apiClient.Get(ctx, requestedJobID)
	if err != nil {
		spinner.Done(false)
		Fatal(cmd, err.Error(), 1)
		return nil
	}

	if !jobFound {
		spinner.Done(false)
	}

	// Check status to make sure there is something to be canceled. If it is currently
	// in a terminal state, then we'll exit immediately
	if job.State.State.IsTerminal() {
		spinner.Done(false)
		errorMessage := fmt.Sprintf(jobAlreadyCompleteMessage, job.State.State.String())
		Fatal(cmd, errorMessage, 1)
		return nil
	}

	// Submit a request to cancel the specified job. It is the responsibility of the
	// requester to decide if we are allowed to do that or not.
	spinner.NextStep(cancellingJobMessage)

	jobState, err := apiClient.Cancel(ctx, job.Job.Metadata.ID, "Canceled at user request")
	if err != nil {
		spinner.Done(false)

		if er, ok := err.(*bacerrors.ErrorResponse); ok {
			Fatal(cmd, er.Error(), 1)
			return nil
		} else {
			Fatal(cmd, fmt.Sprintf("Unknown error trying to cancel job (ID: %s): %+v", requestedJobID, err), 1)
			return nil
		}
	}

	spinner.Done(true)
	cmd.Printf("Job successfully canceled. Job ID: %s\n", jobState.JobID)

	return nil
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

package job

import (
	"fmt"
	"io"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/printer"
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
)

var (
	stopLong = templates.LongDesc(i18n.T(`
		Stop a previously submitted job.
`))

	//nolint:lll // Documentation
	stopExample = templates.Examples(i18n.T(`
		# Stop a previously submitted job
		bacalhau job stop j-51225160-807e-48b8-88c9-28311c7899e1

		# Stop a job, with a short ID.
		bacalhau job stop j-51225160
`))
)

var (
	checkingJobStatusMessage = i18n.T("Checking job status")
	connectingMessage        = i18n.T("Connecting to network")
	gettingJobMessage        = i18n.T("Verifying job state")
	stoppingJobMessage       = i18n.T("Stoping job")

	jobAlreadyCompleteMessage = i18n.T(`Job is already in a terminal state.
The current state is: %s
`)
)

type StopOptions struct {
	Quiet bool
}

func NewStopOptions() *StopOptions {
	return &StopOptions{
		Quiet: false,
	}
}

func NewStopCmd() *cobra.Command {
	o := NewStopOptions()

	stopCmd := &cobra.Command{
		Use:     "stop [id]",
		Short:   "Stop a previously submitted job",
		Long:    stopLong,
		Example: stopExample,
		Args:    cobra.ExactArgs(1),
		PreRun:  util.ApplyPorcelainLogLevel,
		RunE:    o.run,
	}

	stopCmd.PersistentFlags().BoolVar(&o.Quiet, "quiet", o.Quiet,
		`Do not print anything to stdout or stderr`,
	)
	return stopCmd
}

func (o *StopOptions) run(cmd *cobra.Command, cmdArgs []string) error {
	ctx := cmd.Context()

	if o.Quiet {
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
	}

	cmd.Printf("%s\n\n", checkingJobStatusMessage)

	widestString := findWidestString(
		checkingJobStatusMessage,
		connectingMessage,
		gettingJobMessage,
		stoppingJobMessage,
	)

	writer := cmd.OutOrStdout()
	if o.Quiet {
		writer = io.Discard
	}
	// Create a spinner that will exit if/when it sees ctrl-c
	spinner, err := printer.NewSpinner(ctx, writer, widestString, true)
	if err != nil {
		return err
	}
	spinner.Run()

	requestedJobID := cmdArgs[0]
	if requestedJobID == "" {
		var byteResult []byte
		byteResult, err = util.ReadFromStdinIfAvailable(cmd)
		if err != nil {
			return fmt.Errorf("unknown error reading from file: %s", err)
		}
		requestedJobID = string(byteResult)
	}

	// Let the user know we are initiating the request
	spinner.NextStep(connectingMessage)
	apiClient := util.GetAPIClientV2(ctx)

	// Fetch the job information so we can check whether the task is already
	// terminal or not. We will not send requests if it is.
	spinner.NextStep(gettingJobMessage)
	response, err := apiClient.Jobs().Get(&apimodels.GetJobRequest{
		JobID: requestedJobID,
	})
	if err != nil {
		spinner.Done(printer.StopFailed)
		return err
	}

	// Check status to make sure there is something to be stopped. If it is currently
	// in a terminal state, then we'll exit immediately
	job := response.Job
	if job.IsTerminal() {
		spinner.Done(printer.StopFailed)
		errorMessage := fmt.Errorf(jobAlreadyCompleteMessage, job.State.StateType.String())
		return errorMessage
	}

	// Submit a request to stop the specified job. It is the responsibility of the
	// requester to decide if we are allowed to do that or not.
	spinner.NextStep(stoppingJobMessage)

	stopResponse, err := apiClient.Jobs().Stop(&apimodels.StopJobRequest{
		JobID:  requestedJobID,
		Reason: "Stopped at user request",
	})
	if err != nil {
		spinner.Done(printer.StopFailed)
		if errResp, ok := err.(*bacerrors.ErrorResponse); ok {
			return errResp
		}
		return fmt.Errorf("unknown error trying to stop job (ID: %s): %w", requestedJobID, err)
	}

	spinner.Done(printer.StopSuccess)
	cmd.Printf("\nJob stop successfully submitted with evaluation ID: %s\n", stopResponse.EvaluationID)

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

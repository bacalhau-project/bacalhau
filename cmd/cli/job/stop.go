package job

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"

	"github.com/bacalhau-project/bacalhau/cmd/util/templates"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/printer"
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
)

var (
	stopLong = templates.LongDesc(`
		Stop a previously submitted job. Either Job ID or Job Name can be specified.
`)

	stopExample = templates.Examples(`
		# Stop a previously submitted job using Job Name (default namespace)
		bacalhau job stop my-job-name-a

		# Stop a previously submitted job using Job Name and non default namespace
		bacalhau job stop my-job-name-b --namespace=dev

		# Stop a previously submitted job using Job ID (long ID version)
		bacalhau job stop j-51225160-807e-48b8-88c9-28311c7899e1

		# Stop a job, with a short ID.
		bacalhau job stop j-51225160
`)
)

var (
	checkingJobStatusMessage = "Checking job status"

	connectingMessage  = "Connecting to network"
	gettingJobMessage  = "Verifying job state"
	stoppingJobMessage = "Stopping job"

	jobAlreadyCompleteMessage = `Job is already in a terminal state.
The current state is: %s
`
)

type StopOptions struct {
	Quiet     bool
	Namespace string
}

func NewStopOptions() *StopOptions {
	return &StopOptions{
		Quiet: false,
	}
}

func NewStopCmd() *cobra.Command {
	o := NewStopOptions()

	stopCmd := &cobra.Command{
		Use:           "stop",
		Short:         "Stop a previously submitted job",
		Long:          stopLong,
		Example:       stopExample,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// initialize a new or open an existing repo merging any config file(s) it contains into cfg.
			cfg, err := util.SetupRepoConfig(cmd)
			if err != nil {
				return fmt.Errorf("failed to setup repo: %w", err)
			}
			// create an api client
			api, err := util.NewAPIClientManager(cmd, cfg).GetAuthenticatedAPIClient()
			if err != nil {
				return fmt.Errorf("failed to create api client: %w", err)
			}
			return o.run(cmd, args, api)
		},
	}

	stopCmd.SilenceUsage = true
	stopCmd.SilenceErrors = true

	stopCmd.PersistentFlags().BoolVar(&o.Quiet, "quiet", o.Quiet,
		`Do not print anything to stdout or stderr`,
	)
	stopCmd.PersistentFlags().StringVar(&o.Namespace, "namespace", o.Namespace,
		`Job Namespace. If not provided, it will be treated as default namespace.`,
	)
	return stopCmd
}

func (o *StopOptions) run(cmd *cobra.Command, cmdArgs []string, api client.API) error {
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

	requestedJobIdentifier := cmdArgs[0]
	if requestedJobIdentifier == "" {
		var byteResult []byte
		byteResult, err = util.ReadFromStdinIfAvailable(cmd)
		if err != nil {
			return fmt.Errorf("unknown error reading from file: %s", err)
		}
		requestedJobIdentifier = string(byteResult)
	}

	// Let the user know we are initiating the request
	spinner.NextStep(connectingMessage)
	// Fetch the job information so we can check whether the task is already
	// terminal or not. We will not send requests if it is.
	spinner.NextStep(gettingJobMessage)
	response, err := api.Jobs().Get(ctx, &apimodels.GetJobRequest{
		JobIDOrName: requestedJobIdentifier,
		BaseGetRequest: apimodels.BaseGetRequest{
			BaseRequest: apimodels.BaseRequest{
				Namespace: o.Namespace,
			},
		},
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

	stopResponse, err := api.Jobs().Stop(ctx, &apimodels.StopJobRequest{
		JobID:  requestedJobIdentifier,
		Reason: "Stopped at user request",
		BasePutRequest: apimodels.BasePutRequest{
			BaseRequest: apimodels.BaseRequest{
				Namespace: o.Namespace,
			},
		},
	})
	if err != nil {
		spinner.Done(printer.StopFailed)
		if bacerrors.IsError(err) {
			return err
		}
		return fmt.Errorf("unknown error trying to stop job (ID: %s): %w", requestedJobIdentifier, err)
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

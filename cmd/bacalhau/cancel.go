package bacalhau

import (
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
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

type CancelOptions struct{}

func NewCancelOptions() *CancelOptions {
	return &CancelOptions{}
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

	return cancelCmd
}

func cancel(cmd *cobra.Command, cmdArgs []string, options *CancelOptions) error {
	ctx := cmd.Context()

	var err error

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

	apiClient := GetAPIClient()

	// Submit a request to cancel the specified job. It is the responsibility of the
	// requester to decide if we are allowed to do that or not.
	jobState, err := apiClient.Cancel(ctx, requestedJobID, "Canceled at user request")
	if err != nil {
		if er, ok := err.(*bacerrors.ErrorResponse); ok {
			Fatal(cmd, er.Message, 1)
			return nil
		} else {
			Fatal(cmd, fmt.Sprintf("Unknown error trying to cancel job (ID: %s): %+v", requestedJobID, err), 1)
			return nil
		}
	}

	cmd.Printf("Job successfully canceled. Job ID: %s\n", jobState.JobID)

	return nil
}

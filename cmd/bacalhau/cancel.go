package bacalhau

import (
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
	"github.com/filecoin-project/bacalhau/pkg/telemetry"

	"github.com/filecoin-project/bacalhau/pkg/system"
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
	cm := system.NewCleanupManager()
	defer cm.Cleanup()
	ctx := cmd.Context()

	ctx, span := system.NewRootSpan(ctx, system.GetTracer(), "cmd/bacalhau.cancel")
	defer span.End()
	cm.RegisterCallback(telemetry.Cleanup)

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

	// Retrieve information about the job with the matching Job ID. No checks are
	// made here about ownership of the job as that responsibility sits with the
	// requester, and it will decide whether we get the job info or not, and
	// eventually whether we can cancel the job or not.
	_, foundJob, err := apiClient.Get(ctx, requestedJobID)
	if err != nil {
		if er, ok := err.(*bacerrors.ErrorResponse); ok {
			Fatal(cmd, er.Message, 1)
			return nil
		} else {
			Fatal(cmd, fmt.Sprintf("Unknown error trying to retrieve job info (ID: %s): %+v", requestedJobID, err), 1)
			return nil
		}
	}

	if !foundJob {
		cmd.Printf(err.Error() + "\n")
		Fatal(cmd, "", 1)
	}

	// Check actual status of job (no need to cancel if complete) and then
	// submit request to cancel.

	return nil
}

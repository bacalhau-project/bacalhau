package hook

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/version"
	"github.com/spf13/cobra"
)

var printMessage *string = nil

// StartUpdateCheck is a Cobra pre run hook to run an update check in the
// background. There should be no output if the check fails or the context is
// cancelled before the check can complete.
func StartUpdateCheck(cmd *cobra.Command, args []string) {
	version.RunUpdateChecker(
		cmd.Context(),
		func(ctx context.Context) (*models.BuildVersionInfo, error) {
			if response, err := util.GetAPIClientV2().Agent().Version(ctx); err != nil {
				return nil, err
			} else if response != nil {
				return response.BuildVersionInfo, nil
			} else {
				return nil, nil
			}
		},
		func(_ context.Context, ucr *version.UpdateCheckResponse) { printMessage = &ucr.Message },
	)
}

// PrintUpdateCheck is a Cobra post run hook to print the results of an update
// check. The message will be a non-nil pointer only if the update check
// succeeds and should only have visible output if the message is non-empty.
func PrintUpdateCheck(cmd *cobra.Command, args []string) {
	if printMessage != nil && *printMessage != "" {
		fmt.Fprintln(cmd.ErrOrStderr())
		fmt.Fprintln(cmd.ErrOrStderr(), *printMessage)
	}
}

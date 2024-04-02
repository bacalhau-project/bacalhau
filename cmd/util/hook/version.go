package hook

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
	"github.com/bacalhau-project/bacalhau/pkg/version"
	"github.com/spf13/cobra"
)

var printMessage *string = nil

// StartUpdateCheck is a Cobra pre run hook to run an update check in the
// background. There should be no output if the check fails or the context is
// cancelled before the check can complete.
func StartUpdateCheck(cmd *cobra.Command, args []string) {
	legacyTLS := client.LegacyTLSSupport(config.ClientTLSConfig())

	version.RunUpdateChecker(
		cmd.Context(),
		client.NewAPIClient(legacyTLS, config.ClientAPIHost(), config.ClientAPIPort()).Version,
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

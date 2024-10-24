package util

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

var Fatal = fatalError

func fatalError(cmd *cobra.Command, err error, code int) {
	cmd.PrintErrln()

	var bErr bacerrors.Error
	if errors.As(err, &bErr) {
		// Print error message
		cmd.PrintErrln(output.RedStr("Error: ") + bErr.Error())

		// Print hint if available
		if bErr.Hint() != "" {
			cmd.PrintErrln(output.GreenStr("Hint:  ") + bErr.Hint())
		}

		// If debug mode, then print details and stack trace
		if system.IsDebugMode() {
			if len(bErr.Details()) > 0 {
				cmd.PrintErrln()
				cmd.PrintErrln(output.YellowStr("Details:"))
				for k, v := range bErr.Details() {
					cmd.PrintErrln(k + ": " + v)
				}
			}
			stackTrace := bErr.StackTrace()
			if stackTrace != "" {
				cmd.PrintErrln()
				cmd.PrintErrln(output.YellowStr("Stack Trace:"))
				cmd.PrintErrln(stackTrace)
			}
		}
	} else {
		cmd.PrintErrln(output.RedStr("Error: ") + err.Error())
	}
	os.Exit(code)
}

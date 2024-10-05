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
	cmd.SetOut(os.Stdout)
	cmd.Println()

	var bErr bacerrors.Error
	if errors.As(err, &bErr) {
		// Print error message
		cmd.Println(output.RedStr("Error: ") + bErr.Error())

		// Print hint if available
		if bErr.Hint() != "" {
			cmd.Println(output.GreenStr("Hint:  ") + bErr.Hint())
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
				cmd.Println()
				cmd.Println(output.YellowStr("Stack Trace:"))
				cmd.Println(stackTrace)
			}
		}

	} else {
		cmd.Println(output.RedStr("Error: ") + err.Error())
	}
	os.Exit(code)
}

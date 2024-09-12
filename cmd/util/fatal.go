package util

import (
	"os"
	"strings"

	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/spf13/cobra"
)

var Fatal = fatalError

func fatalError(cmd *cobra.Command, err error, code int) {
	if msg := err.Error(); msg != "" {
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		cmd.PrintErr(output.RedStr(msg))
	}
	os.Exit(code)
}

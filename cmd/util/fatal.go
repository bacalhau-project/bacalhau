package util

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var Fatal = fatalError

func fatalError(cmd *cobra.Command, err error, code int) {
	if msg := err.Error(); msg != "" {
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		cmd.PrintErr(msg)
	}
	os.Exit(code)
}

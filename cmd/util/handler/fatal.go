package handler

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var Fatal = fatalError

func fatalError(cmd *cobra.Command, err error, code int) {
	if msg := err.Error(); msg != "" {
		// add newline if needed
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		cmd.Print(msg)
	}
	os.Exit(code)
}

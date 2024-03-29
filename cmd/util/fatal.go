package util

import (
	"errors"
	"os"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/lib/bad"
	"github.com/spf13/cobra"
)

var Fatal = fatalError

func fatalError(cmd *cobra.Command, err error, code int) {
	defer os.Exit(code)

	if err == nil {
		return
	}

	apiErr := new(bad.Error)
	if errors.As(err, &apiErr) {
		PrintErr(cmd, apiErr)
	} else if msg := err.Error(); msg != "" {
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		cmd.PrintErr(msg)
	}
}

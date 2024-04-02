package util

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func ReadFromStdinIfAvailable(cmd *cobra.Command) ([]byte, error) {
	// write to stderr since stdout is reserved for details of jobs.
	if _, err := io.WriteString(os.Stderr, "Reading from /dev/stdin; send Ctrl-d to stop."); err != nil {
		return nil, fmt.Errorf("unable to write to stderr")
	}
	return io.ReadAll(cmd.InOrStdin())
}

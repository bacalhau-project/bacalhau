package testutils

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func ExecuteCommandC(t *testing.T, root *cobra.Command, args ...string) (*cobra.Command, string, error, *zap.Logger) {

	l := zaptest.NewLogger(t)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	// Need to check if we're running in debug mode for VSCode
	// Empty them if they exist
	if (len(os.Args) > 2) && (os.Args[1] == "-test.run") {
		os.Args[1] = ""
		os.Args[2] = ""
	}

	l.Debug(fmt.Sprintf("Command to execute: bacalhau %v", root.CalledAs()))

	c, err := root.ExecuteC()
	return c, buf.String(), err, l
}

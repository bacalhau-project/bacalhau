package bacalhau

import (
	"bytes"
	"io"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func ExecuteTestCobraCommand(args ...string) (c *cobra.Command, output string, err error) {
	return ExecuteTestCobraCommandWithStdin(nil, args...)
}

func ExecuteTestCobraCommandWithStdin(stdin io.Reader, args ...string) (c *cobra.Command, output string, err error) {
	buf := new(bytes.Buffer)
	root := NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(stdin)
	root.SetArgs(args)

	// Need to check if we're running in debug mode for VSCode
	// Empty them if they exist
	if (len(os.Args) > 2) && (os.Args[1] == "-test.run") {
		os.Args[1] = ""
		os.Args[2] = ""
	}

	log.Trace().Msgf("Command to execute: %v", root.CalledAs())

	c, err = root.ExecuteC()
	return c, buf.String(), err
}

package utils

import (
	"bytes"
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	log "go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)
func ExecuteCommandC(cmd *cobra.Command, logger *log.Logger, args ...string) (c *cobra.Command, output string, err error) {
	buf := new(bytes.Buffer)

	root := cmd.Root()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	// // Need to check if we're running in debug mode for VSCode
	// // Empty them if they exist
	// if (len(os.Args) > 2) && (os.Args[1] == "-test.run") {
	// 	os.Args[1] = ""
	// 	os.Args[2] = ""
	// }

	logger.Debug(fmt.Sprintf("Command to execute: bacalhau %v", root.CalledAs()))

	c, err = root.ExecuteC()
	return c, buf.String(), err
}


func SetupLogsCapture() (*zap.Logger, *observer.ObservedLogs) {
    core, logs := observer.New(zap.InfoLevel)
    return zap.New(core), logs
}

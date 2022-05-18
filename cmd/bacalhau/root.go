package bacalhau

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var jsonrpcPort int
var jsonrpcHost string

func init() {
	RootCmd.AddCommand(serveCmd)
	RootCmd.AddCommand(runCmd)
	RootCmd.AddCommand(listCmd)
	RootCmd.AddCommand(resultsCmd)
	RootCmd.AddCommand(devstackCmd)
	RootCmd.PersistentFlags().IntVar(
		&jsonrpcPort, "jsonrpc-port", 1234,
		`The port for the client and server to communicate on (via jsonrpc).`,
	)
	RootCmd.PersistentFlags().StringVar(
		&jsonrpcHost, "jsonrpc-host", "0.0.0.0",
		`The port for the client and server to communicate on (via jsonrpc).`,
	)
}

var RootCmd = &cobra.Command{
	Use:   "bacalhau",
	Short: "Compute over data",
	Long:  `Compute over data`,
}

func Execute(version string) {
	RootCmd.Version = version

	setVersion()

	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func setVersion() {
	template := fmt.Sprintf("Bacalhau Version: %s\n", RootCmd.Version)
	RootCmd.SetVersionTemplate(template)
}

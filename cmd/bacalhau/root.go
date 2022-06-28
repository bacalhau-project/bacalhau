package bacalhau

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var apiHost string
var apiPort int
var DefaultPortNumber = 1234

func init() { // nolint:gochecknoinits // Using init in cobra command is idomatic
	RootCmd.AddCommand(serveCmd)
	RootCmd.AddCommand(dockerCmd)
	RootCmd.AddCommand(getCmd)
	RootCmd.AddCommand(listCmd)
	RootCmd.AddCommand(describeCmd)
	RootCmd.AddCommand(devstackCmd)
	RootCmd.PersistentFlags().StringVar(
		&apiHost, "api-host", "bootstrap.production.bacalhau.org",
		`The host for the client and server to communicate on (via REST).`,
	)
	RootCmd.PersistentFlags().IntVar(
		&apiPort, "api-port", DefaultPortNumber,
		`The port for the client and server to communicate on (via REST).`,
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

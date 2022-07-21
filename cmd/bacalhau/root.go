package bacalhau

import (
	"fmt"
	"os"

	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/spf13/cobra"
)

var apiHost string
var apiPort int

func init() { // nolint:gochecknoinits // Using init in cobra command is idomatic
	RootCmd.AddCommand(serveCmd)

	// Porcelain commands (language specific easy to use commands)
	RootCmd.AddCommand(runCmd)

	// Plumbing commands (advanced usage)
	RootCmd.AddCommand(dockerCmd)
	// TODO: RootCmd.AddCommand(wasmCmd)
	RootCmd.AddCommand(applyCmd)

	RootCmd.AddCommand(getCmd)
	RootCmd.AddCommand(listCmd)
	RootCmd.AddCommand(describeCmd)
	RootCmd.AddCommand(devstackCmd)
	RootCmd.PersistentFlags().StringVar(
		&apiHost, "api-host", system.Envs[system.Production].APIHost,
		`The host for the client and server to communicate on (via REST).`,
	)
	RootCmd.PersistentFlags().IntVar(
		&apiPort, "api-port", system.Envs[system.Production].APIPort,
		`The port for the client and server to communicate on (via REST).`,
	)
	RootCmd.AddCommand(versionCmd)
}

var RootCmd = &cobra.Command{
	Use:   "bacalhau",
	Short: "Compute over data",
	Long:  `Compute over data`,
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

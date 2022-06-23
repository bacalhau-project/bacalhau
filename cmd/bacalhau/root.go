package bacalhau

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var apiHost string
var apiPort int

func init() {
	RootCmd.AddCommand(serveCmd)

	// Porcelain commands (language specific easy to use commands)
	// TODO: RootCmd.AddCommand(runCmd)

	// Plumbing commands (advanced usage)
	RootCmd.AddCommand(dockerCmd)
	// TODO: RootCmd.AddCommand(wasmCmd)

	RootCmd.AddCommand(getCmd)
	RootCmd.AddCommand(listCmd)
	RootCmd.AddCommand(devstackCmd)
	RootCmd.PersistentFlags().StringVar(
		&apiHost, "api-host", "bootstrap.production.bacalhau.org",
		`The host for the client and server to communicate on (via REST).`,
	)
	RootCmd.PersistentFlags().IntVar(
		&apiPort, "api-port", 1234,
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

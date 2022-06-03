package bacalhau

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var apiHost string
var apiPort int

func init() {
	RootCmd.AddCommand(serveCmd)
	RootCmd.AddCommand(runCmd)
	RootCmd.AddCommand(listCmd)
	RootCmd.AddCommand(devstackCmd)
	RootCmd.PersistentFlags().StringVar(
		&apiHost, "api-host", "http://bootstrap.production.bacalhau.org",
		`The host for the client and server to communicate on (via REST).`,
	)

	if !strings.Contains(apiHost, "http://") && !strings.Contains(apiHost, "https://") {
		log.Fatal().Msgf("You must prefix your api-host address with 'http://' or 'http://'. Provided: %s", apiHost)
	}

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

package bacalhau

import (
	"fmt"
	"os"
	"strconv"

	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		`The host for the client and server to communicate on (via REST). Ignored if BACALHAU_API_HOST environment variable is set.`,
	)
	RootCmd.PersistentFlags().IntVar(
		&apiPort, "api-port", system.Envs[system.Production].APIPort,
		`The port for the client and server to communicate on (via REST). Ignored if BACALHAU_API_PORT environment variable is set.`,
	)
	RootCmd.AddCommand(versionCmd)
}

var RootCmd = &cobra.Command{
	Use:   "bacalhau",
	Short: "Compute over data",
	Long:  `Compute over data`,
}

func Execute() {
	viper.SetEnvPrefix("BACALHAU")

	err := viper.BindEnv("API_HOST")
	if err != nil {
		log.Fatal().Msgf("API_HOST was set, but could not bind.")
	}

	err = viper.BindEnv("API_PORT")
	if err != nil {
		log.Fatal().Msgf("API_PORT was set, but could not bind.")
	}

	viper.AutomaticEnv()
	envAPIHost := viper.Get("API_HOST")
	envAPIPort := viper.Get("API_PORT")

	if envAPIHost != nil && envAPIHost != "" {
		apiHost = envAPIHost.(string)
	}

	if envAPIPort != nil && envAPIPort != "" {
		var parseErr error
		apiPort, parseErr = strconv.Atoi(envAPIPort.(string))
		if parseErr != nil {
			log.Fatal().Msgf("could not parse API_PORT into an int. %s", envAPIPort)
		}
	}

	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

package bacalhau

import (
	"context"
	"os"
	"strconv"

	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var apiHost string
var apiPort int
var doNotTrack bool

var Fatal = FatalErrorHandler

func init() { //nolint:gochecknoinits // Using init in cobra command is idomatic
	// ====== Start a job

	// Create job from file
	RootCmd.AddCommand(createCmd)

	// Plumbing commands (advanced usage)
	RootCmd.AddCommand(dockerCmd)
	RootCmd.AddCommand(wasmCmd)

	// Porcelain commands (language specific easy to use commands)
	RootCmd.AddCommand(runCmd)

	RootCmd.AddCommand(validateCmd)

	RootCmd.AddCommand(versionCmd)

	// ====== Get information or results about a job
	// Describe a job
	RootCmd.AddCommand(describeCmd)

	// Get the results of a job
	RootCmd.AddCommand(getCmd)

	// List jobs
	RootCmd.AddCommand(listCmd)

	// ====== Run a server

	// Serve commands
	RootCmd.AddCommand(serveCmd)
	RootCmd.AddCommand(simulatorCmd)
	RootCmd.AddCommand(idCmd)
	RootCmd.AddCommand(devstackCmd)

	defaultAPIHost := system.Envs[system.Production].APIHost
	defaultAPIPort := system.Envs[system.Production].APIPort

	if config.GetAPIHost() != "" {
		defaultAPIHost = config.GetAPIHost()
	}

	if config.GetAPIPort() != "" {
		intPort, err := strconv.Atoi(config.GetAPIPort())
		if err == nil {
			defaultAPIPort = intPort
		}
	}

	RootCmd.PersistentFlags().StringVar(
		&apiHost, "api-host", defaultAPIHost,
		`The host for the client and server to communicate on (via REST).
Ignored if BACALHAU_API_HOST environment variable is set.`,
	)
	RootCmd.PersistentFlags().IntVar(
		&apiPort, "api-port", defaultAPIPort,
		`The port for the client and server to communicate on (via REST).
Ignored if BACALHAU_API_PORT environment variable is set.`,
	)
}

var RootCmd = &cobra.Command{
	Use:   getCommandLineExecutable(),
	Short: "Compute over data",
	Long:  `Compute over data`,
}

func Execute() {
	// ANCHOR: Set global context here
	RootCmd.SetContext(context.Background())

	doNotTrack = false
	if doNotTrackValue, foundDoNotTrack := os.LookupEnv("DO_NOT_TRACK"); foundDoNotTrack {
		doNotTrackInt, err := strconv.Atoi(doNotTrackValue)
		if err == nil && doNotTrackInt == 1 {
			doNotTrack = true
		}
	}

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

	// Use stdout, not stderr for cmd.Print output, so that
	// e.g. ID=$(bacalhau run) works
	RootCmd.SetOutput(system.Stdout)

	if err := RootCmd.Execute(); err != nil {
		Fatal(err.Error(), 1)
	}
}

package bacalhau

import (
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var apiHost string
var apiPort int
var doNotTrack bool

var loggingMode logger.Logmode = logger.LogModeDefault

var Fatal = FatalErrorHandler

var defaultAPIHost string
var defaultAPIPort int

func init() { //nolint:gochecknoinits
	defaultAPIHost = system.Envs[system.GetEnvironment()].APIHost
	defaultAPIPort = system.Envs[system.GetEnvironment()].APIPort

	if config.GetAPIHost() != "" {
		defaultAPIHost = config.GetAPIHost()
	}

	if config.GetAPIPort() != "" {
		intPort, err := strconv.Atoi(config.GetAPIPort())
		if err == nil {
			defaultAPIPort = intPort
		}
	}

	if logtype, set := os.LookupEnv("LOG_TYPE"); set {
		loggingMode = logger.Logmode(strings.ToLower(logtype))
	}

	// Force cobra to set apiHost & apiPort
	NewRootCmd()
}

func NewRootCmd() *cobra.Command {
	RootCmd := &cobra.Command{
		Use:   getCommandLineExecutable(),
		Short: "Compute over data",
		Long:  `Compute over data`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			logger.ConfigureLogging(loggingMode)
		},
	}
	// ====== Start a job

	// Create job from file
	RootCmd.AddCommand(newCreateCmd())

	// Plumbing commands (advanced usage)
	RootCmd.AddCommand(newDockerCmd())
	RootCmd.AddCommand(newWasmCmd())

	// Porcelain commands (language specific easy to use commands)
	RootCmd.AddCommand(newRunCmd())

	RootCmd.AddCommand(newValidateCmd())

	RootCmd.AddCommand(newVersionCmd())

	// ====== Get information or results about a job
	// Describe a job
	RootCmd.AddCommand(newDescribeCmd())

	// Get the results of a job
	RootCmd.AddCommand(newGetCmd())

	// Cancel a job
	RootCmd.AddCommand(newCancelCmd())

	// List jobs
	RootCmd.AddCommand(newListCmd())

	// ====== Run a server

	// Serve commands
	RootCmd.AddCommand(newServeCmd())
	RootCmd.AddCommand(newSimulatorCmd())
	RootCmd.AddCommand(newIDCmd())
	RootCmd.AddCommand(newDevStackCmd())

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
	RootCmd.PersistentFlags().Var(
		LoggingFlag(&loggingMode), "log-mode",
		`Log format: 'default','station','json','combined','event'`,
	)
	return RootCmd
}

func Execute() {
	RootCmd := NewRootCmd()
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
		Fatal(RootCmd, err.Error(), 1)
	}
}

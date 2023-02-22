package bacalhau

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/telemetry"
	"github.com/filecoin-project/bacalhau/pkg/version"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/trace"
)

var apiHost string
var apiPort int
var doNotTrack bool

var loggingMode = logger.LogModeDefault

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
		loggingMode = logger.LogMode(strings.ToLower(logtype))
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
			ctx := cmd.Context()

			logger.ConfigureLogging(loggingMode)

			cm := system.NewCleanupManager()
			cm.RegisterCallback(telemetry.Cleanup)
			ctx = context.WithValue(ctx, systemManagerKey, cm)

			var names []string
			root := cmd
			for ; root.HasParent(); root = root.Parent() {
				names = append([]string{root.Name()}, names...)
			}
			name := fmt.Sprintf("bacalhau.%s", strings.Join(names, "."))
			ctx, span := system.NewRootSpan(ctx, system.GetTracer(), name)
			ctx = context.WithValue(ctx, spanKey, span)

			cmd.SetContext(ctx)
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			ctx.Value(spanKey).(trace.Span).End()
			ctx.Value(systemManagerKey).(*system.CleanupManager).Cleanup(cmd.Context())
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
		log.Ctx(RootCmd.Context()).Fatal().Msgf("API_HOST was set, but could not bind.")
	}

	err = viper.BindEnv("API_PORT")
	if err != nil {
		log.Ctx(RootCmd.Context()).Fatal().Msgf("API_PORT was set, but could not bind.")
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
			log.Ctx(RootCmd.Context()).Fatal().Msgf("could not parse API_PORT into an int. %s", envAPIPort)
		}
	}

	// Use stdout, not stderr for cmd.Print output, so that
	// e.g. ID=$(bacalhau run) works
	RootCmd.SetOut(system.Stdout)
	// TODO this is from fixing a deprecation warning for SetOutput. Shouldn't this be system.Stderr?
	RootCmd.SetErr(system.Stdout)

	if err := RootCmd.Execute(); err != nil {
		Fatal(RootCmd, err.Error(), 1)
	}
}

type contextKey struct {
	name string
}

var systemManagerKey = contextKey{name: "context key for storing the system manager"}
var spanKey = contextKey{name: "context key for storing the root span"}

func checkVersion(cmd *cobra.Command, args []string) error {
	// corba doesn't do PersistentPreRun{,E} chaining yet
	// https://github.com/spf13/cobra/issues/252
	root := cmd
	for ; root.HasParent(); root = root.Parent() {
	}
	root.PersistentPreRun(cmd, args)

	// Check that the server version is compatible with the client version
	serverVersion, _ := GetAPIClient().Version(cmd.Context()) // Ok if this fails, version validation will skip
	if err := ensureValidVersion(cmd.Context(), version.Get(), serverVersion); err != nil {
		Fatal(cmd, fmt.Sprintf("version validation failed: %s", err), 1)
		return err
	}

	return nil
}

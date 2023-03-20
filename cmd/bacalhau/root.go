package bacalhau

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
	"github.com/bacalhau-project/bacalhau/pkg/version"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/trace"
)

var apiHost string
var apiPort uint16

var loggingMode = logger.LogModeDefault

var Fatal = FatalErrorHandler

var defaultAPIHost string
var defaultAPIPort uint16

func init() { //nolint:gochecknoinits
	defaultAPIHost = system.Envs[system.GetEnvironment()].APIHost
	defaultAPIPort = system.Envs[system.GetEnvironment()].APIPort

	if config.GetAPIHost() != "" {
		defaultAPIHost = config.GetAPIHost()
	}

	if config.GetAPIPort() != nil {
		defaultAPIPort = *config.GetAPIPort()
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
			ctx.Value(systemManagerKey).(*system.CleanupManager).Cleanup(ctx)
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

	// Get logs
	RootCmd.AddCommand(newLogsCmd())

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
	RootCmd.PersistentFlags().Uint16Var(
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
	rootCmd := NewRootCmd()

	// Ensure commands are able to stop cleanly if someone presses ctrl+c
	ctx, cancel := signal.NotifyContext(context.Background(), ShutdownSignals...)
	defer cancel()
	rootCmd.SetContext(ctx)

	viper.SetEnvPrefix("BACALHAU")

	if err := viper.BindEnv("API_HOST"); err != nil {
		log.Ctx(ctx).Fatal().Msgf("API_HOST was set, but could not bind.")
	}

	if err := viper.BindEnv("API_PORT"); err != nil {
		log.Ctx(ctx).Fatal().Msgf("API_PORT was set, but could not bind.")
	}

	viper.AutomaticEnv()

	if envAPIHost := viper.GetString("API_HOST"); envAPIHost != "" {
		apiHost = envAPIHost
	}

	if envAPIPort := viper.GetString("API_PORT"); envAPIPort != "" {
		var parseErr error
		parsedPort, parseErr := strconv.ParseUint(envAPIPort, 10, 16)
		if parseErr != nil {
			log.Ctx(ctx).Fatal().Msgf("could not parse API_PORT into an int. %s", envAPIPort)
		} else {
			apiPort = uint16(parsedPort)
		}
	}

	// Use stdout, not stderr for cmd.Print output, so that
	// e.g. ID=$(bacalhau run) works
	rootCmd.SetOut(system.Stdout)
	// TODO this is from fixing a deprecation warning for SetOutput. Shouldn't this be system.Stderr?
	rootCmd.SetErr(system.Stdout)

	if err := rootCmd.Execute(); err != nil {
		Fatal(rootCmd, err.Error(), 1)
	}
}

type contextKey struct {
	name string
}

var systemManagerKey = contextKey{name: "context key for storing the system manager"}
var spanKey = contextKey{name: "context key for storing the root span"}

func checkVersion(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// corba doesn't do PersistentPreRun{,E} chaining yet
	// https://github.com/spf13/cobra/issues/252
	root := cmd
	for ; root.HasParent(); root = root.Parent() {
	}
	root.PersistentPreRun(cmd, args)

	// Check that the server version is compatible with the client version
	serverVersion, _ := GetAPIClient().Version(ctx) // Ok if this fails, version validation will skip
	if err := ensureValidVersion(ctx, version.Get(), serverVersion); err != nil {
		Fatal(cmd, fmt.Sprintf("version validation failed: %s", err), 1)
		return err
	}

	return nil
}

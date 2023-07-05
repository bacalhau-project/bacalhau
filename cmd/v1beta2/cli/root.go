package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/trace"

	"github.com/bacalhau-project/bacalhau/cmd/v1beta2/cli/cancel"
	"github.com/bacalhau-project/bacalhau/cmd/v1beta2/cli/create"
	"github.com/bacalhau-project/bacalhau/cmd/v1beta2/cli/describe"
	"github.com/bacalhau-project/bacalhau/cmd/v1beta2/cli/devstack"
	"github.com/bacalhau-project/bacalhau/cmd/v1beta2/cli/docker"
	"github.com/bacalhau-project/bacalhau/cmd/v1beta2/cli/get"
	"github.com/bacalhau-project/bacalhau/cmd/v1beta2/cli/id"
	"github.com/bacalhau-project/bacalhau/cmd/v1beta2/cli/list"
	"github.com/bacalhau-project/bacalhau/cmd/v1beta2/cli/logs"
	"github.com/bacalhau-project/bacalhau/cmd/v1beta2/cli/nodes"
	"github.com/bacalhau-project/bacalhau/cmd/v1beta2/cli/serve"
	"github.com/bacalhau-project/bacalhau/cmd/v1beta2/cli/simulate"
	"github.com/bacalhau-project/bacalhau/cmd/v1beta2/cli/validate"
	"github.com/bacalhau-project/bacalhau/cmd/v1beta2/cli/version"
	"github.com/bacalhau-project/bacalhau/cmd/v1beta2/cli/wasm"
	util2 "github.com/bacalhau-project/bacalhau/cmd/v1beta2/util"
	"github.com/bacalhau-project/bacalhau/cmd/v1beta2/util/flags"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
)

var apiHost string
var apiPort uint16

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
		util2.LoggingMode = logger.LogMode(strings.ToLower(logtype))
	}

	// Force cobra to set apiHost & apiPort
	NewRootCmd()
}

func NewRootCmd() *cobra.Command {
	RootCmd := &cobra.Command{
		Use:   os.Args[0],
		Short: "Compute over data",
		Long:  `Compute over data`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()

			logger.ConfigureLogging(util2.LoggingMode)

			cm := system.NewCleanupManager()
			cm.RegisterCallback(telemetry.Cleanup)
			ctx = context.WithValue(ctx, util2.SystemManagerKey, cm)

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
			ctx.Value(util2.SystemManagerKey).(*system.CleanupManager).Cleanup(ctx)
		},
	}
	// ====== Start a job

	// Create job from file
	RootCmd.AddCommand(create.NewCmd())

	// Plumbing commands (advanced usage)
	RootCmd.AddCommand(docker.NewCmd())
	RootCmd.AddCommand(wasm.NewCmd())

	RootCmd.AddCommand(validate.NewCmd())

	RootCmd.AddCommand(version.NewCmd())

	// ====== Get information or results about a job
	// Describe a job
	RootCmd.AddCommand(describe.NewCmd())

	// Get logs
	RootCmd.AddCommand(logs.NewCmd())

	// Get the results of a job
	RootCmd.AddCommand(get.NewCmd())

	// Cancel a job
	RootCmd.AddCommand(cancel.NewCmd())

	// List jobs
	RootCmd.AddCommand(list.NewCmd())

	// List nodes
	RootCmd.AddCommand(nodes.NewCmd())

	// ====== Run a server

	// Serve commands
	RootCmd.AddCommand(serve.NewCmd())
	RootCmd.AddCommand(simulate.NewCmd())
	RootCmd.AddCommand(id.NewCmd())
	RootCmd.AddCommand(devstack.NewCmd())

	RootCmd.PersistentFlags().StringVar(
		&apiHost, "api-host", defaultAPIHost,
		`The host for the client and server to communicate on (via REST).
Ignored if BACALHAU_API_HOST environment variable is set.`,
	)
	if err := viper.BindPFlag("api-host", RootCmd.PersistentFlags().Lookup("api-host")); err != nil {
		panic(err)
	}
	RootCmd.PersistentFlags().Uint16Var(
		&apiPort, "api-port", defaultAPIPort,
		`The port for the client and server to communicate on (via REST).
Ignored if BACALHAU_API_PORT environment variable is set.`,
	)
	if err := viper.BindPFlag("api-port", RootCmd.PersistentFlags().Lookup("api-port")); err != nil {
		panic(err)
	}
	RootCmd.PersistentFlags().Var(
		flags.LoggingFlag(&util2.LoggingMode), "log-mode",
		`Log format: 'default','station','json','combined','event'`,
	)
	return RootCmd
}

func Execute() {
	rootCmd := NewRootCmd()

	// Ensure commands are able to stop cleanly if someone presses ctrl+c
	ctx, cancel := signal.NotifyContext(context.Background(), util2.ShutdownSignals...)
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
	rootCmd.SetErr(system.Stderr)

	if err := rootCmd.Execute(); err != nil {
		util2.Fatal(rootCmd, err, 1)
	}
}

type contextKey struct {
	name string
}

var spanKey = contextKey{name: "context key for storing the root span"}

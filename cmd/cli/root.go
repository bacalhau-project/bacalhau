package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/trace"

	"github.com/bacalhau-project/bacalhau/cmd/cli/auth"

	"github.com/bacalhau-project/bacalhau/cmd/cli/agent"
	configcli "github.com/bacalhau-project/bacalhau/cmd/cli/config"
	"github.com/bacalhau-project/bacalhau/cmd/cli/devstack"
	"github.com/bacalhau-project/bacalhau/cmd/cli/docker"
	"github.com/bacalhau-project/bacalhau/cmd/cli/job"
	"github.com/bacalhau-project/bacalhau/cmd/cli/node"
	"github.com/bacalhau-project/bacalhau/cmd/cli/profile"
	"github.com/bacalhau-project/bacalhau/cmd/cli/serve"
	"github.com/bacalhau-project/bacalhau/cmd/cli/version"
	"github.com/bacalhau-project/bacalhau/cmd/cli/wasm"
	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	"github.com/bacalhau-project/bacalhau/pkg/common"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
)

func NewRootCmd() *cobra.Command {
	// the root command: `bacalhau`
	RootCmd := &cobra.Command{
		Use:   os.Args[0],
		Short: "Compute over data",
		Long:  `Compute over data`,
	}

	RootCmd.PersistentFlags().VarP(cliflags.NewConfigFlag(), "config", "c", "config file(s) or dot separated path(s) to config values")
	if err := RootCmd.RegisterFlagCompletionFunc("config", cliflags.ConfigAutoComplete); err != nil {
		util.Fatal(RootCmd, err, 1)
	}

	// flag definitions with a corresponding field in the config file.
	// when these flags are provided their value will be used instead of the value present in the config file.
	// If no flg is provided, and the config file doesn't have a value defined then the default value will be used.
	rootFlags := map[string][]configflags.Definition{
		"api":     configflags.ClientAPIFlags,
		"logging": configflags.LogFlags,
		"repo":    configflags.DataDirFlag,
	}
	// register the flags on the command.
	if err := configflags.RegisterFlags(RootCmd, rootFlags); err != nil {
		// a failure here indicates a developer error, abort.
		util.Fatal(RootCmd, err, 1)
	}

	// Add global profile flag (no shorthand to avoid conflict with -p for publisher in run commands)
	RootCmd.PersistentFlags().String("profile", "", "Use a specific profile for this command")

	// logic that must run before any child command executes
	RootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// context of the command
		ctx := cmd.Context()
		ctx = util.InjectCleanupManager(ctx)
		ctx = injectRootSpan(cmd, ctx)

		// Profile selection - store flag and env var in context
		profileFlagValue, _ := cmd.Flags().GetString("profile")
		profileEnvValue := os.Getenv("BACALHAU_PROFILE")
		ctx = context.WithValue(ctx, util.ProfileFlagKey, profileFlagValue)
		ctx = context.WithValue(ctx, util.ProfileEnvKey, profileEnvValue)

		cmd.SetContext(ctx)

		// Binds flags with a corresponding config file value to the root command.
		// This is done in the pre-run so their values are set in the Run function.
		// Cobra doesn't allow the root pre run method to run if a child command also has a prerun defined.
		if err := configflags.BindFlags(viper.GetViper(), rootFlags); err != nil {
			return err
		}

		// Configure logging
		// While we allow users to configure logging via the config file, they are applied
		// and will override this configuration at a later stage when the config is loaded.
		// This is needed to ensure any logs before the config is loaded are captured.
		logLevel := viper.GetString(types.LoggingLevelKey)
		if logLevel == "" {
			logLevel = "Info"
		}
		if err := logger.ParseAndConfigureLogging(string(logger.LogModeCmd), logLevel); err != nil {
			return fmt.Errorf("failed to configure logging: %w", err)
		}

		return nil
	}

	// logic that must run after any child command completes.
	RootCmd.PersistentPostRunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		ctx.Value(spanKey).(trace.Span).End()
		ctx.Value(util.SystemManagerKey).(*system.CleanupManager).Cleanup(ctx)
		return nil
	}

	// register child commands.
	RootCmd.AddCommand(
		agent.NewCmd(),
		configcli.NewCmd(),
		devstack.NewCmd(),
		docker.NewCmd(),
		job.NewCmd(),
		auth.NewCmd(),
		node.NewCmd(),
		profile.NewCmd(),
		serve.NewCmd(),
		version.NewCmd(),
		wasm.NewCmd(),
	)

	// Customize help template to include environment variables section
	helpTemplate := RootCmd.HelpTemplate() + fmt.Sprintf(`
Auth Environment Variables:
  %s         API key for builtin authentication
  %s    Username for Basic Auth builtin authentication
  %s    Password for Basic Auth builtin authentication
`, common.BacalhauAPIKey, common.BacalhauAPIUsername, common.BacalhauAPIPassword)
	RootCmd.SetHelpTemplate(helpTemplate)

	return RootCmd
}

func Execute(ctx context.Context) {
	rootCmd := NewRootCmd()
	rootCmd.SetContext(ctx)
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true

	// this is needed as cobra defaults to Stderr if no output is set.
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	if err := rootCmd.Execute(); err != nil {
		util.Fatal(rootCmd, err, 1)
	}
}

type contextKey struct {
	name string
}

var spanKey = contextKey{name: "context key for storing the root span"}

func injectRootSpan(cmd *cobra.Command, ctx context.Context) context.Context {
	var names []string
	root := cmd
	for ; root.HasParent(); root = root.Parent() {
		names = append([]string{root.Name()}, names...)
	}
	name := fmt.Sprintf("bacalhau.%s", strings.Join(names, "."))
	ctx, span := telemetry.NewRootSpan(ctx, telemetry.GetTracer(), name)
	return context.WithValue(ctx, spanKey, span)
}

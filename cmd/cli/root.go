package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/trace"

	"github.com/bacalhau-project/bacalhau/cmd/cli/agent"
	"github.com/bacalhau-project/bacalhau/cmd/cli/deprecated"
	"github.com/bacalhau-project/bacalhau/cmd/cli/exec"
	"github.com/bacalhau-project/bacalhau/cmd/cli/job"
	"github.com/bacalhau-project/bacalhau/cmd/cli/node"

	configcli "github.com/bacalhau-project/bacalhau/cmd/cli/config"
	"github.com/bacalhau-project/bacalhau/cmd/cli/devstack"
	"github.com/bacalhau-project/bacalhau/cmd/cli/docker"
	"github.com/bacalhau-project/bacalhau/cmd/cli/serve"
	"github.com/bacalhau-project/bacalhau/cmd/cli/version"
	"github.com/bacalhau-project/bacalhau/cmd/cli/wasm"
	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
)

//nolint:funlen
func NewRootCmd() *cobra.Command {
	rootFlags := map[string][]configflags.Definition{
		"api":     configflags.ClientAPIFlags,
		"logging": configflags.LogFlags,
	}
	cfgViper := viper.GetViper()
	RootCmd := &cobra.Command{
		Use:   os.Args[0],
		Short: "Compute over data",
		Long:  `Compute over data`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := configflags.BindFlags(cmd, cfgViper, rootFlags); err != nil {
				util.Fatal(cmd, err, 1)
			}

			logger.ConfigureLogging(util.LoggingMode)

			cm := system.NewCleanupManager()
			cm.RegisterCallback(telemetry.Cleanup)
			ctx := context.WithValue(cmd.Context(), util.SystemManagerKey, cm)

			var names []string
			root := cmd
			for ; root.HasParent(); root = root.Parent() {
				names = append([]string{root.Name()}, names...)
			}
			name := fmt.Sprintf("bacalhau.%s", strings.Join(names, "."))
			ctx, span := system.NewRootSpan(ctx, system.GetTracer(), name)
			ctx = context.WithValue(ctx, spanKey, span)

			cmd.SetContext(ctx)
			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			ctx.Value(spanKey).(trace.Span).End()
			ctx.Value(util.SystemManagerKey).(*system.CleanupManager).Cleanup(ctx)
			return nil
		},
	}
	// ensure the `repo` key always gets a usable default value, warn if it's not.
	defaultRepo, err := defaultRepo()
	if err != nil {
		RootCmd.Printf("WARNING: %s\n"+
			"cannot determine default repo location: "+
			"BACALHAU_DIR or --repo must be set to initialize a node.\n\n", err)
	}
	RootCmd.PersistentFlags().String("repo", defaultRepo, "path to bacalhau repo")

	// Bind the repo flag to the system configuration
	if err := cfgViper.BindPFlag("repo", RootCmd.PersistentFlags().Lookup("repo")); err != nil {
		util.Fatal(RootCmd, err, 1)
	}
	if err := cfgViper.BindEnv("repo", "BACALHAU_DIR"); err != nil {
		util.Fatal(RootCmd, err, 1)
	}

	if err := configflags.RegisterFlags(RootCmd, rootFlags); err != nil {
		panic(err)
	}

	// ====== Start a job
	RootCmd.AddCommand(deprecated.NewCreateCmd())

	// Plumbing commands (advanced usage)
	RootCmd.AddCommand(docker.NewCmd())
	RootCmd.AddCommand(wasm.NewCmd())

	RootCmd.AddCommand(deprecated.NewValidateCmd())

	RootCmd.AddCommand(version.NewCmd())

	// ====== Get information or results about a job
	// Describe a job
	RootCmd.AddCommand(deprecated.NewDescribeCmd())

	// Get logs
	RootCmd.AddCommand(deprecated.NewLogsCmd())

	// Get the results of a job
	RootCmd.AddCommand(deprecated.NewGetCmd())

	// Cancel a job
	RootCmd.AddCommand(deprecated.NewCancelCmd())

	// List jobs
	RootCmd.AddCommand(deprecated.NewListCmd())

	// Register agent subcommands
	RootCmd.AddCommand(agent.NewCmd())

	// Register job subcommands
	RootCmd.AddCommand(job.NewCmd())

	// Register nodes subcommands
	RootCmd.AddCommand(node.NewCmd())

	// Register exec commands
	RootCmd.AddCommand(exec.NewCmd())

	// ====== Run a server

	// Serve commands
	RootCmd.AddCommand(serve.NewCmd())
	RootCmd.AddCommand(deprecated.NewIDCmd())
	RootCmd.AddCommand(devstack.NewCmd())

	// config command...obviously
	RootCmd.AddCommand(configcli.NewCmd())

	return RootCmd
}

func Execute(ctx context.Context) {
	rootCmd := NewRootCmd()
	rootCmd.SetContext(ctx)

	// Use stdout, not stderr for cmd.Print output, so that
	// e.g. ID=$(bacalhau run) works
	rootCmd.SetOut(system.Stdout)
	rootCmd.SetErr(system.Stderr)

	if err := rootCmd.Execute(); err != nil {
		util.Fatal(rootCmd, err, 1)
	}
}

type contextKey struct {
	name string
}

var spanKey = contextKey{name: "context key for storing the root span"}

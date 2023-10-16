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
	"github.com/bacalhau-project/bacalhau/cmd/cli/job"
	"github.com/bacalhau-project/bacalhau/cmd/cli/node"

	"github.com/bacalhau-project/bacalhau/cmd/cli/cancel"
	"github.com/bacalhau-project/bacalhau/cmd/cli/create"
	"github.com/bacalhau-project/bacalhau/cmd/cli/describe"
	"github.com/bacalhau-project/bacalhau/cmd/cli/devstack"
	"github.com/bacalhau-project/bacalhau/cmd/cli/docker"
	"github.com/bacalhau-project/bacalhau/cmd/cli/get"
	"github.com/bacalhau-project/bacalhau/cmd/cli/id"
	"github.com/bacalhau-project/bacalhau/cmd/cli/list"
	"github.com/bacalhau-project/bacalhau/cmd/cli/logs"
	"github.com/bacalhau-project/bacalhau/cmd/cli/serve"
	"github.com/bacalhau-project/bacalhau/cmd/cli/validate"
	"github.com/bacalhau-project/bacalhau/cmd/cli/version"
	"github.com/bacalhau-project/bacalhau/cmd/cli/wasm"
	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
)

//nolint:funlen
func NewRootCmd() *cobra.Command {
	rootFlags := map[string][]configflags.Definition{
		"api":     configflags.ClientAPIFlags,
		"logging": configflags.LogFlags,
	}
	RootCmd := &cobra.Command{
		Use:     os.Args[0],
		Short:   "Compute over data",
		Long:    `Compute over data`,
		PreRun:  util.StartUpdateCheck,
		PostRun: util.PrintUpdateCheck,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			repoDir, err := config.Get[string]("repo")
			if err != nil {
				util.Fatal(cmd, fmt.Errorf("failed to read --repo value: %w", err), 1)
			}
			if repoDir == "" {
				// this error indicates `defaultRepo` was unable to find a default location and the user
				// didn't provide on.
				util.Fatal(cmd, fmt.Errorf("bacalhau repo not set, please use BACALHAU_DIR or --repo"), 1)
			}
			if _, err := setup.SetupBacalhauRepo(repoDir); err != nil {
				util.Fatal(cmd, fmt.Errorf("failed to initialize bacalhau repo at '%s': %w", repoDir, err), 1)
			}

			if err := configflags.BindFlags(cmd, rootFlags); err != nil {
				util.Fatal(cmd, err, 1)
			}

			ctx := cmd.Context()

			logger.ConfigureLogging(util.LoggingMode)

			cm := system.NewCleanupManager()
			cm.RegisterCallback(telemetry.Cleanup)
			ctx = context.WithValue(ctx, util.SystemManagerKey, cm)

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
	if err := viper.BindPFlag("repo", RootCmd.PersistentFlags().Lookup("repo")); err != nil {
		util.Fatal(RootCmd, err, 1)
	}
	if err := viper.BindEnv("repo", "BACALHAU_DIR"); err != nil {
		util.Fatal(RootCmd, err, 1)
	}

	if err := configflags.RegisterFlags(RootCmd, rootFlags); err != nil {
		panic(err)
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

	// Register agent subcommands
	RootCmd.AddCommand(agent.NewCmd())

	// Register job subcommands
	RootCmd.AddCommand(job.NewCmd())

	// Register nodes subcommands
	RootCmd.AddCommand(node.NewCmd())

	// ====== Run a server

	// Serve commands
	RootCmd.AddCommand(serve.NewCmd())
	RootCmd.AddCommand(id.NewCmd())
	RootCmd.AddCommand(devstack.NewCmd())

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

package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/trace"

	"github.com/bacalhau-project/bacalhau/cmd/cli/agent"
	"github.com/bacalhau-project/bacalhau/cmd/cli/exec"
	"github.com/bacalhau-project/bacalhau/cmd/cli/job"
	"github.com/bacalhau-project/bacalhau/cmd/cli/node"
	"github.com/bacalhau-project/bacalhau/pkg/config"

	"github.com/bacalhau-project/bacalhau/cmd/cli/cancel"
	configcli "github.com/bacalhau-project/bacalhau/cmd/cli/config"
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
	cfg := config.New()
	RootCmd := &cobra.Command{
		Use:   os.Args[0],
		Short: "Compute over data",
		Long:  `Compute over data`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// TODO(forrest) [correctness]: we need a way to bind these flags to the config in the child commands
			// of the root method so that when the config is loaded it either uses the values provided to these flag
			// or in the event they are not present, it uses the default value, or value found in the config file.
			if err := configflags.BindFlagsWithViper(cmd, cfg.Viper(), rootFlags); err != nil {
				util.Fatal(cmd, err, 1)
			}

			// TODO(forrest) [refactor]: decide how we want to handle this case
			// ideally it comes via a validate method during configuration initialization
			/*
				// If a CA certificate was provided, it must be a file that exists. If it does not
				// exist we should not continue.
				if caCert, err := config.Get[string](types.NodeClientAPIClientTLSCACert); err == nil && caCert != "" {
					if _, err := os.Stat(caCert); os.IsNotExist(err) {
						util.Fatal(cmd, fmt.Errorf("CA certificate file '%s' does not exist", caCert), 1)
					}
				}
			*/

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
	defaultRepo, err := DefaultRepo()
	if err != nil {
		RootCmd.Printf("WARNING: %s\n"+
			"cannot determine default repo location: "+
			"BACALHAU_DIR or --repo must be set to initialize a node.\n\n", err)
	}
	RootCmd.PersistentFlags().String("repo", defaultRepo, "path to bacalhau repo")

	// Bind the repo flag to the system configuration
	if err := cfg.System().BindPFlag("repo", RootCmd.PersistentFlags().Lookup("repo")); err != nil {
		util.Fatal(RootCmd, err, 1)
	}
	if err := cfg.System().BindEnv("repo", "BACALHAU_DIR"); err != nil {
		util.Fatal(RootCmd, err, 1)
	}

	if err := configflags.RegisterFlags(RootCmd, rootFlags); err != nil {
		panic(err)
	}

	// ====== Start a job

	// Create job from file
	RootCmd.AddCommand(create.NewCmd(cfg))

	// Plumbing commands (advanced usage)
	RootCmd.AddCommand(docker.NewCmd(cfg))
	RootCmd.AddCommand(wasm.NewCmd(cfg))

	RootCmd.AddCommand(validate.NewCmd())

	RootCmd.AddCommand(version.NewCmd())

	// ====== Get information or results about a job
	// Describe a job
	RootCmd.AddCommand(describe.NewCmd(cfg))

	// Get logs
	RootCmd.AddCommand(logs.NewCmd(cfg))

	// Get the results of a job
	RootCmd.AddCommand(get.NewCmd(cfg))

	// Cancel a job
	RootCmd.AddCommand(cancel.NewCmd(cfg))

	// List jobs
	RootCmd.AddCommand(list.NewCmd(cfg))

	// Register agent subcommands
	RootCmd.AddCommand(agent.NewCmd(cfg))

	// Register job subcommands
	RootCmd.AddCommand(job.NewCmd(cfg))

	// Register nodes subcommands
	RootCmd.AddCommand(node.NewCmd(cfg))

	// Register exec commands
	RootCmd.AddCommand(exec.NewCmd(cfg))

	// ====== Run a server

	// Serve commands
	RootCmd.AddCommand(serve.NewCmd(cfg))
	RootCmd.AddCommand(id.NewCmd(cfg))
	RootCmd.AddCommand(devstack.NewCmd())

	// config command...obviously
	RootCmd.AddCommand(configcli.NewCmd(cfg))

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

package devstack

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/viper"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/setup"

	"github.com/bacalhau-project/bacalhau/cmd/cli/serve"
	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"

	"github.com/spf13/cobra"
)

var (
	devStackLong = templates.LongDesc(i18n.T(`
		Start a cluster of nodes and run a job on them.
`))

	//nolint:lll // Documentation
	devstackExample = templates.Examples(i18n.T(`
		# Create a devstack cluster with a single requester node and 3 compute nodes (Default values)
		bacalhau devstack

		# Create a devstack cluster with a two requester nodes and 10 compute nodes
		bacalhau devstack  --requester-nodes 2 --compute-nodes 10

		# Create a devstack cluster with a single hybrid (requester and compute) nodes
		bacalhau devstack  --requester-nodes 0 --compute-nodes 0 --hybrid-nodes 1

		# Run a devstack and create (or use) the config repo in a specific folder
		bacalhau devstack  --stack-repo ./my-devstack-configuration
`))
)

func newDevStackOptions() *devstack.DevStackOptions {
	return &devstack.DevStackOptions{
		NumberOfRequesterOnlyNodes: 1,
		NumberOfComputeOnlyNodes:   3,
		NumberOfBadComputeActors:   0,
		Peer:                       "",
		CPUProfilingFile:           "",
		MemoryProfilingFile:        "",
		ConfigurationRepo:          "",
	}
}

//nolint:funlen,gocyclo
func NewCmd() *cobra.Command {
	ODs := newDevStackOptions()
	IsNoop := false
	devstackFlags := map[string][]configflags.Definition{
		"publishing":            configflags.PublishingFlags,
		"requester-tls":         configflags.RequesterTLSFlags,
		"job-selection":         configflags.JobSelectionFlags,
		"disable-features":      configflags.DisabledFeatureFlags,
		"capacity":              configflags.CapacityFlags,
		"job-timeouts":          configflags.ComputeTimeoutFlags,
		"translations":          configflags.JobTranslationFlags,
		"docker-cache-manifest": configflags.DockerManifestCacheFlags,
	}

	devstackCmd := &cobra.Command{
		Use:     "devstack",
		Short:   "Start a cluster of bacalhau nodes for testing and development",
		Long:    devStackLong,
		Example: devstackExample,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return configflags.BindFlags(cmd, viper.GetViper(), devstackFlags)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			// ensure we either use a temp repo for the devstack, or the repo path provided
			// by the specific devstack flag. Never use the default bacalhau repo.
			v := viper.GetViper()
			repoPath := ODs.ConfigurationRepo
			if repoPath == "" {
				// We need to clean up the repo when the node shuts down, but we can ONLY
				// do this because we know it is a temporary directory. Do not delete the
				// configured repo if `--stack-repo` was specified
				repoPath, _ = os.MkdirTemp("", "")

				// If we don't set the repo value in config, readers will be given
				// a different path to the one we've just created. Presumably a default.
				defer os.RemoveAll(repoPath)
			}
			// override the repo path set in the root command with the derived path.
			v.Set("repo", repoPath)
			cfg := config.New(config.WithViper(v))
			// create or open the bacalhau repo and load the config
			fsr, err := setup.SetupBacalhauRepo(repoPath, cfg)
			if err != nil {
				return fmt.Errorf("failed to reconcile repo: %w", err)
			}
			bcfg, err := cfg.Current()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			return runDevstack(cmd, bcfg, fsr, ODs, IsNoop)
		},
	}

	if err := configflags.RegisterFlags(devstackCmd, devstackFlags); err != nil {
		util.Fatal(devstackCmd, err, 1)
	}

	devstackCmd.PersistentFlags().IntVar(
		&ODs.NumberOfHybridNodes, "hybrid-nodes", ODs.NumberOfHybridNodes,
		`How many hybrid (requester and compute) nodes should be started in the cluster`,
	)
	devstackCmd.PersistentFlags().IntVar(
		&ODs.NumberOfRequesterOnlyNodes, "requester-nodes", ODs.NumberOfRequesterOnlyNodes,
		`How many requester only nodes should be started in the cluster`,
	)
	devstackCmd.PersistentFlags().IntVar(
		&ODs.NumberOfComputeOnlyNodes, "compute-nodes", ODs.NumberOfComputeOnlyNodes,
		`How many compute only nodes should be started in the cluster`,
	)
	devstackCmd.PersistentFlags().IntVar(
		&ODs.NumberOfBadComputeActors, "bad-compute-actors", ODs.NumberOfBadComputeActors,
		`How many compute nodes should be bad actors`,
	)
	devstackCmd.PersistentFlags().IntVar(
		&ODs.NumberOfBadRequesterActors, "bad-requester-actors", ODs.NumberOfBadRequesterActors,
		`How many requester nodes should be bad actors`,
	)
	devstackCmd.PersistentFlags().BoolVar(
		&IsNoop, "Noop", false,
		`Use the noop executor for all jobs`,
	)
	devstackCmd.PersistentFlags().StringVar(
		&ODs.Peer, "peer", ODs.Peer,
		`Connect node 0 to another network node`,
	)
	devstackCmd.PersistentFlags().StringVar(
		&ODs.CPUProfilingFile, "cpu-profiling-file", ODs.CPUProfilingFile,
		"File to save CPU profiling to",
	)
	devstackCmd.PersistentFlags().StringVar(
		&ODs.MemoryProfilingFile, "memory-profiling-file", ODs.MemoryProfilingFile,
		"File to save memory profiling to",
	)
	devstackCmd.PersistentFlags().StringSliceVar(
		&ODs.AllowListedLocalPaths, "allow-listed-local-paths", ODs.AllowListedLocalPaths,
		"Local paths that are allowed to be mounted into jobs. Multiple paths can be specified by using this flag multiple times.",
	)
	devstackCmd.PersistentFlags().BoolVar(
		&ODs.ExecutorPlugins, "pluggable-executors", ODs.ExecutorPlugins,
		"Will use pluggable executors when set to true",
	)
	devstackCmd.PersistentFlags().StringVar(
		&ODs.ConfigurationRepo, "stack-repo", ODs.ConfigurationRepo,
		"Folder to act as the devstack configuration repo",
	)
	return devstackCmd
}

//nolint:gocyclo,funlen
func runDevstack(cmd *cobra.Command, cfg types.BacalhauConfig, fsr *repo.FsRepo, ODs *devstack.DevStackOptions, IsNoop bool) error {
	ctx := cmd.Context()

	cm := util.GetCleanupManager(ctx)
	cm.RegisterCallback(telemetry.Cleanup)

	config.DevstackSetShouldPrintInfo()

	portFileName := filepath.Join(os.TempDir(), "bacalhau-devstack.port")
	pidFileName := filepath.Join(os.TempDir(), "bacalhau-devstack.pid")

	if _, ignore := os.LookupEnv("IGNORE_PID_AND_PORT_FILES"); !ignore {
		_, err := os.Stat(portFileName)
		if err == nil {
			return fmt.Errorf("found file %s - Devstack likely already running", portFileName)
		}
		_, err = os.Stat(pidFileName)
		if err == nil {
			return fmt.Errorf("found file %s - Devstack likely already running", pidFileName)
		}
	}

	computeConfig, err := serve.GetComputeConfig(ctx, cfg.Node, true)
	if err != nil {
		return err
	}

	requesterConfig, err := serve.GetRequesterConfig(ctx, cfg.Node.Requester, true)
	if err != nil {
		return err
	}

	options := append(ODs.Options(),
		devstack.WithComputeConfig(computeConfig),
		devstack.WithRequesterConfig(requesterConfig),
	)
	if IsNoop {
		options = append(options, devstack.WithDependencyInjector(devstack.NewNoopNodeDependencyInjector()))
	} else if ODs.ExecutorPlugins {
		options = append(options, devstack.WithDependencyInjector(node.NewExecutorPluginNodeDependencyInjector(cfg)))
	} else {
		options = append(options, devstack.WithDependencyInjector(node.NewStandardNodeDependencyInjector(cfg)))
	}

	// Get any certificate settings for devstack and use them if we have a certificate (possibly self-signed).
	options = append(options, devstack.WithSelfSignedCertificate(
		cfg.Node.ServerAPI.TLS.ServerCertificate,
		cfg.Node.ServerAPI.TLS.ServerKey,
	))

	stack, err := devstack.Setup(ctx, cfg, cm, fsr, options...)
	if err != nil {
		return err
	}

	nodeInfoOutput, err := stack.PrintNodeInfo(ctx, fsr, cm)
	if err != nil {
		return fmt.Errorf("failed to print node info: %w", err)
	}
	cmd.Println(nodeInfoOutput)

	f, err := os.Create(portFileName)
	if err != nil {
		return fmt.Errorf("error writing out port file to %v: %w", portFileName, err)
	}
	defer os.Remove(portFileName)
	firstNode := stack.Nodes[0]
	_, err = f.WriteString(strconv.FormatUint(uint64(firstNode.APIServer.Port), 10))
	if err != nil {
		return fmt.Errorf("error writing out port file: %v: %w", portFileName, err)
	}

	fPid, err := os.Create(pidFileName)
	if err != nil {
		return fmt.Errorf("error writing out pid file to %v: %w", pidFileName, err)
	}
	defer os.Remove(pidFileName)

	_, err = fPid.WriteString(strconv.Itoa(os.Getpid()))
	if err != nil {
		return fmt.Errorf("error writing out pid file: %v: %w", pidFileName, err)
	}

	<-ctx.Done() // block until killed

	cmd.Println("\nShutting down devstack")
	return nil
}

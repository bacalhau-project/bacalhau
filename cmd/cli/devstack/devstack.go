package devstack

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config_legacy"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/webui"

	"k8s.io/kubectl/pkg/util/templates"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"

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

type options struct {
	NumberOfHybridNodes        int // Number of nodes to start in the cluster
	NumberOfRequesterOnlyNodes int // Number of nodes to start in the cluster
	NumberOfComputeOnlyNodes   int // Number of nodes to start in the cluster
	NumberOfBadComputeActors   int // Number of compute nodes to be bad actors
	CPUProfilingFile           string
	MemoryProfilingFile        string
	BasePath                   string
	WebUIListen                string
}

func (o *options) devstackOptions() []devstack.ConfigOption {
	opts := []devstack.ConfigOption{
		devstack.WithNumberOfHybridNodes(o.NumberOfHybridNodes),
		devstack.WithNumberOfRequesterOnlyNodes(o.NumberOfRequesterOnlyNodes),
		devstack.WithNumberOfComputeOnlyNodes(o.NumberOfComputeOnlyNodes),
		devstack.WithNumberOfBadComputeActors(o.NumberOfBadComputeActors),
		devstack.WithCPUProfilingFile(o.CPUProfilingFile),
		devstack.WithMemoryProfilingFile(o.MemoryProfilingFile),
		devstack.WithBasePath(o.BasePath),
	}
	return opts
}

func newOptions() *options {
	return &options{
		NumberOfRequesterOnlyNodes: 1,
		NumberOfComputeOnlyNodes:   3,
		NumberOfBadComputeActors:   0,
		CPUProfilingFile:           "",
		MemoryProfilingFile:        "",
		BasePath:                   "",
		WebUIListen:                config.Default.WebUI.Listen,
	}
}

//nolint:funlen,gocyclo
func NewCmd() *cobra.Command {
	ODs := newOptions()
	devstackFlags := map[string][]configflags.Definition{
		"job-selection":         configflags.JobSelectionFlags,
		"disable-features":      configflags.DisabledFeatureFlags,
		"capacity":              configflags.CapacityFlags,
		"translations":          configflags.JobTranslationFlags,
		"docker-cache-manifest": configflags.DockerManifestCacheFlags,
	}

	devstackCmd := &cobra.Command{
		Use:     "devstack",
		Short:   "Start a cluster of bacalhau nodes for testing and development",
		Long:    devStackLong,
		Example: devstackExample,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return configflags.BindFlags(viper.GetViper(), devstackFlags)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			// TODO: a hack to force debug logging for devstack
			//  until I figure out why flags and env vars are not working
			logger.ConfigureLogging(logger.LogModeDefault, zerolog.DebugLevel)
			return runDevstack(cmd, ODs)
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
	devstackCmd.PersistentFlags().StringVar(
		&ODs.WebUIListen, "webui-address", ODs.WebUIListen,
		`Listen address for the web UI server`,
	)
	devstackCmd.PersistentFlags().StringVar(
		&ODs.CPUProfilingFile, "cpu-profiling-file", ODs.CPUProfilingFile,
		"File to save CPU profiling to",
	)
	devstackCmd.PersistentFlags().StringVar(
		&ODs.MemoryProfilingFile, "memory-profiling-file", ODs.MemoryProfilingFile,
		"File to save memory profiling to",
	)
	devstackCmd.PersistentFlags().StringVar(
		&ODs.BasePath, "stack-repo", ODs.BasePath,
		"Folder to act as the devstack configuration repo",
	)
	return devstackCmd
}

//nolint:gocyclo,funlen
func runDevstack(cmd *cobra.Command, ODs *options) error {
	ctx := cmd.Context()

	cm := util.GetCleanupManager(ctx)
	cm.RegisterCallback(telemetry.Cleanup)

	config_legacy.DevstackSetShouldPrintInfo()

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

	// ensure we either use a temp repo for the devstack, or the repo path provided
	// by the specific devstack flag. Never use the default bacalhau repo.
	baseRepoPath := ODs.BasePath
	if baseRepoPath == "" {
		// We need to clean up the repo when the node shuts down, but we can ONLY
		// do this because we know it is a temporary directory. Do not delete the
		// configured repo if `--stack-repo` was specified
		baseRepoPath, _ = os.MkdirTemp("", "")
		defer os.RemoveAll(baseRepoPath)
	}

	stack, err := devstack.Setup(ctx, cm, ODs.devstackOptions()...)
	if err != nil {
		return err
	}

	// start WebUI for the first successful requester node
	for _, n := range stack.Nodes {
		// TODO: move webui creation to node pkg
		if n.IsRequesterNode() {
			webuiConfig := webui.Config{
				APIEndpoint: n.APIServer.GetURI().String(),
				Listen:      ODs.WebUIListen,
			}
			webuiServer, err := webui.NewServer(webuiConfig)
			if err != nil {
				log.Warn().Err(err).Msg("Failed to start ui server for this node, trying next")
				continue
			}

			go func() {
				if err := webuiServer.ListenAndServe(ctx); err != nil {
					log.Error().Err(err).Msg("ui server error")
				}
			}()

			break
		}
	}

	nodeInfoOutput, err := stack.PrintNodeInfo(ctx, cm)
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

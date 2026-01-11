package serve

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/templates"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/analytics"
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/lib/network"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/version"
	"github.com/bacalhau-project/bacalhau/webui"
)

var (
	serveLong = templates.LongDesc(`
		Start a bacalhau node.
		`)

	serveExample = templates.Examples(`
		# Start a private bacalhau requester node
		bacalhau serve
		# or
		bacalhau serve --config Orchestrator.Enabled

		# Start a private bacalhau hybrid node that acts as both compute and requester
		bacalhau serve --config Orchestrator.Enabled --config Compute.Enabled
		# or

		# Start a public bacalhau node with the WebUI on port 3000 (default:0.0.0.0:8483)
		bacalhau serve --config WebUI.Enabled --config WebUI.Listen=0.0.0.0:3000
`)
)

const (
	NameFlagName        = "name"
	NameFlagDescription = `The node's name.
If unset, it will be read from .bacalhau/system_metadata.yaml, or automatically generated if no name exists.
If set, and a name isn't present in .bacalhau/system_metadata.yaml the value is persisted, else ignored.`
)

func NewCmd() *cobra.Command {
	serveFlags := map[string][]configflags.Definition{
		"serve": configflags.ServeFlags,
	}
	serveCmd := &cobra.Command{
		Use:           "serve",
		Short:         "Start the bacalhau compute node",
		Long:          serveLong,
		Example:       serveExample,
		SilenceUsage:  true,
		SilenceErrors: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return configflags.BindFlags(viper.GetViper(), serveFlags)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, rawCfg, err := util.SetupConfigs(cmd)
			if err != nil {
				return fmt.Errorf("failed to setup config: %w", err)
			}

			if err = logger.ParseAndConfigureLogging(cfg.Logging.Mode, cfg.Logging.Level); err != nil {
				return fmt.Errorf("failed to configure logging: %w", err)
			}

			log.Info().Msgf("Config loaded from: %s, and with data-dir %s",
				rawCfg.Paths(), rawCfg.Get(types.DataDirKey))

			// create or open the bacalhau repo and load the config
			fsr, err := setup.SetupBacalhauRepo(cfg)
			if err != nil {
				return fmt.Errorf("failed to reconcile repo: %w", err)
			}
			return serve(cmd, cfg, fsr)
		},
	}

	serveCmd.PersistentFlags().String(NameFlagName, "", NameFlagDescription)

	if err := configflags.RegisterFlags(serveCmd, serveFlags); err != nil {
		util.Fatal(serveCmd, err, 1)
	}
	return serveCmd
}

//nolint:funlen
func serve(cmd *cobra.Command, cfg types.Bacalhau, fsRepo *repo.FsRepo) error {
	ctx := cmd.Context()
	cm := util.GetCleanupManager(ctx)

	sysmeta, err := fsRepo.SystemMetadata()
	if err != nil {
		return fmt.Errorf("failed to get system metadata from repo: %w", err)
	}
	if sysmeta.NodeName == "" {
		// Check if a flag was provided
		sysmeta.NodeName = cmd.PersistentFlags().Lookup(NameFlagName).Value.String()
		if sysmeta.NodeName == "" {
			// No flag provided, generate and persist node name
			sysmeta.NodeName, err = config.GenerateNodeID(ctx, cfg.NameProvider)
			if err != nil {
				return fmt.Errorf("failed to generate node name for provider %s: %w", cfg.NameProvider, err)
			}
		}
		// Persist the node name
		if err := fsRepo.WriteNodeName(sysmeta.NodeName); err != nil {
			return fmt.Errorf("failed to write node name %s: %w", sysmeta.NodeName, err)
		}
		log.Info().Msgf("persisted node name %s", sysmeta.NodeName)
		// now reload the system metadata since it has changed.
		sysmeta, err = fsRepo.SystemMetadata()
		if err != nil {
			return fmt.Errorf("reloading system metadata after persisting name: %w", err)
		}
	} else {
		// Warn if the flag was provided but node name already exists
		if flagNodeName := cmd.PersistentFlags().Lookup(NameFlagName).Value.String(); flagNodeName != "" && flagNodeName != sysmeta.NodeName {
			log.Warn().Msgf("--name flag with value %s ignored. Name %s already exists", flagNodeName, sysmeta.NodeName)
		}
	}

	ctx = logger.ContextWithNodeIDLogger(ctx, sysmeta.NodeName)

	if !cfg.Compute.Enabled && !cfg.Orchestrator.Enabled {
		log.Warn().Msg("neither --compute nor --orchestrator were provided, defaulting to orchestrator node.")
		cfg.Orchestrator.Enabled = true
	}

	// Create node config from cmd arguments
	// TODO: validate if this is necessary
	hostAddress, err := parseServerAPIHost(cfg.API.Host)
	if err != nil {
		return err
	}
	cfg.API.Host = hostAddress

	nodeConfig := node.NodeConfig{
		NodeID:         sysmeta.NodeName,
		CleanupManager: cm,
		BacalhauConfig: cfg,
	}

	// Create node
	log.Info().Msg("Starting bacalhau...")
	standardNode, err := node.NewNode(ctx, nodeConfig, fsRepo)
	if err != nil {
		return bacerrors.Wrap(err, "failed to start node")
	}

	// Start node
	if err := standardNode.Start(ctx); err != nil {
		return fmt.Errorf("error starting node: %w", err)
	}

	// Start up Dashboard - default: 8483
	if cfg.WebUI.Enabled {
		webuiConfig := webui.Config{
			APIEndpoint: cfg.WebUI.Backend,
			Listen:      cfg.WebUI.Listen,
		}
		if webuiConfig.APIEndpoint == "" {
			webuiConfig.APIEndpoint = standardNode.APIServer.GetURI().String()
		}
		webuiServer, err := webui.NewServer(webuiConfig)
		if err != nil {
			// not failing the node if the webui server fails to start
			log.Error().Err(err).Msg("Failed to start ui server")
		} else {
			go func() {
				if err := webuiServer.ListenAndServe(ctx); err != nil {
					log.Error().Err(err).Msg("ui server error")
				}
			}()
		}
	}

	if !cfg.DisableAnalytics {
		err = analytics.Setup(
			analytics.WithNodeID(sysmeta.NodeName),
			analytics.WithInstallationID(system.InstallationID()),
			analytics.WithInstanceID(sysmeta.InstanceID),
			analytics.WithNodeType(cfg.Orchestrator.Enabled, cfg.Compute.Enabled),
			analytics.WithVersion(version.Get()),
			analytics.WithSystemInfo(),
		)

		if err != nil {
			log.Trace().Err(err).Msg("failed to setup analytics provider")
		} else {
			defer analytics.Shutdown()
		}
	}

	isDebug := system.IsDebugMode()
	startupLog := log.Info().
		Str("name", sysmeta.NodeName)

	if isDebug {
		startupLog.
			Str("address", fmt.Sprintf("%s:%d", hostAddress, cfg.API.Port)).
			Bool("compute_enabled", cfg.Compute.Enabled).
			Bool("orchestrator_enabled", cfg.Orchestrator.Enabled).
			Bool("webui_enabled", cfg.WebUI.Enabled)
	}
	if cfg.Compute.Enabled {
		startupLog.Strs("orchestrators", cfg.Compute.Orchestrators)

		if isDebug {
			capacity := standardNode.ComputeNode.Capacity.GetMaxCapacity(ctx)
			startupLog.
				Strs("engines", standardNode.ComputeNode.Executors.Keys(ctx)).
				Strs("publishers", standardNode.ComputeNode.Publishers.Keys(ctx)).
				Strs("storages", standardNode.ComputeNode.Storages.Keys(ctx)).
				Str("capacity", capacity.String())
			if len(cfg.Compute.AllowListedLocalPaths) > 0 {
				startupLog.Strs("volumes", cfg.Compute.AllowListedLocalPaths)
			}
		}
	}

	if cfg.Orchestrator.Enabled {
		if isDebug {
			startupLog.Str("orchestrator_address",
				fmt.Sprintf("%s:%d", cfg.Orchestrator.Host, cfg.Orchestrator.Port))
		}
	}
	startupLog.Msg("bacalhau node running")

	envvars := buildEnvVariables(cfg)
	riPath, err := fsRepo.WriteRunInfo(ctx, envvars)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(riPath) }()

	<-ctx.Done() // block until killed
	log.Info().Msg("bacalhau node shutting down...")
	return nil
}

func parseServerAPIHost(host string) (string, error) {
	if net.ParseIP(host) == nil {
		// We should check that the value gives us an address type
		// we can use to get our IP address. If it doesn't, we should
		// panic.
		addrType, ok := network.AddressTypeFromString(host)
		if !ok {
			return "", fmt.Errorf("invalid address type in Server API Host config: %s", host)
		}

		addr, err := network.GetNetworkAddress(addrType, network.AllAddresses)
		if err != nil {
			return "", fmt.Errorf("failed to get network address for Server API Host: %s: %w", host, err)
		}

		if len(addr) == 0 {
			return "", fmt.Errorf("no %s addresses found for Server API Host", host)
		}

		// Use the first address
		host = addr[0]
	}

	return host, nil
}

func buildEnvVariables(
	cfg types.Bacalhau,
) string {
	// build shell variables to connect to this node
	var envvars strings.Builder
	envvars.WriteString(fmt.Sprintf("export %s=%s\n", config.KeyAsEnvVar(types.APIHostKey), getAPIURL(cfg.API)))
	envvars.WriteString(fmt.Sprintf("export %s=%d\n", config.KeyAsEnvVar(types.APIPortKey), cfg.API.Port))
	return envvars.String()
}

func getAPIURL(cfg types.API) string {
	if cfg.Host == "0.0.0.0" {
		return "127.0.0.1"
	} else {
		return cfg.Host
	}
}

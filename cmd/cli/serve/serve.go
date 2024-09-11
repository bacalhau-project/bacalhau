package serve

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/lib/crypto"
	"github.com/bacalhau-project/bacalhau/pkg/lib/network"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
	webui "github.com/bacalhau-project/bacalhau/webui_bk"
)

var (
	serveLong = templates.LongDesc(i18n.T(`
		Start a bacalhau node.
		`))

	serveExample = templates.Examples(i18n.T(`
		# Start a private bacalhau requester node
		bacalhau serve
		# or
		bacalhau serve --config Orchestrator.Enabled

		# Start a private bacalhau hybrid node that acts as both compute and requester
		bacalhau serve --config Orchestrator.Enabled --config Compute.Enabled
		# or

		# Start a public bacalhau node with the WebUI on port 3000 (default:0.0.0.0:8483)
		bacalhau serve --config WebUI.Enabled --config WebUI.Listen=0.0.0.0:3000
`))
)

func NewCmd() *cobra.Command {
	serveFlags := map[string][]configflags.Definition{
		"local_publisher":       configflags.LocalPublisherFlags,
		"publishing":            configflags.PublishingFlags,
		"requester-tls":         configflags.RequesterTLSFlags,
		"server-api":            configflags.ServerAPIFlags,
		"network":               configflags.NetworkFlags,
		"ipfs":                  configflags.IPFSFlags,
		"capacity":              configflags.CapacityFlags,
		"job-timeouts":          configflags.ComputeTimeoutFlags,
		"job-selection":         configflags.JobSelectionFlags,
		"disable-features":      configflags.DisabledFeatureFlags,
		"labels":                configflags.LabelFlags,
		"node-type":             configflags.NodeTypeFlags,
		"list-local":            configflags.AllowListLocalPathsFlags,
		"compute-store":         configflags.ComputeStorageFlags,
		"requester-store":       configflags.RequesterJobStorageFlags,
		"web-ui":                configflags.WebUIFlags,
		"node-info-store":       configflags.NodeInfoStoreFlags,
		"node-name":             configflags.NodeNameFlags,
		"translations":          configflags.JobTranslationFlags,
		"docker-cache-manifest": configflags.DockerManifestCacheFlags,
		"compute":               configflags.ComputeFlags,
	}
	serveCmd := &cobra.Command{
		Use:     "serve",
		Short:   "Start the bacalhau compute node",
		Long:    serveLong,
		Example: serveExample,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return configflags.BindFlags(viper.GetViper(), serveFlags)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := util.SetupConfig(cmd)
			if err != nil {
				return fmt.Errorf("failed to setup config: %w", err)
			}
			// create or open the bacalhau repo and load the config
			fsr, err := setup.SetupBacalhauRepo(cfg)
			if err != nil {
				return fmt.Errorf("failed to reconcile repo: %w", err)
			}

			return serve(cmd, cfg, fsr)
		},
	}

	if err := configflags.RegisterFlags(serveCmd, serveFlags); err != nil {
		util.Fatal(serveCmd, err, 1)
	}
	return serveCmd
}

//nolint:funlen,gocyclo
func serve(cmd *cobra.Command, cfg types.Bacalhau, fsRepo *repo.FsRepo) error {
	ctx := cmd.Context()
	cm := util.GetCleanupManager(ctx)

	nodeName, err := fsRepo.ReadNodeName()
	if err != nil {
		return fmt.Errorf("failed to get node name: %w", err)
	}
	ctx = logger.ContextWithNodeIDLogger(ctx, nodeName)

	// configure node type
	isRequesterNode := cfg.Orchestrator.Enabled
	isComputeNode := cfg.Compute.Enabled

	if !(isComputeNode || isRequesterNode) {
		log.Warn().Msg("neither --compute nor --orchestrator were provided, defaulting to orchestrator node.")
		isRequesterNode = true
	}

	networkConfig, err := getNetworkConfig(cfg)
	if err != nil {
		return err
	}

	computeConfig, err := GetComputeConfig(ctx, cfg, isComputeNode)
	if err != nil {
		return errors.Wrapf(err, "failed to configure compute node")
	}

	requesterConfig, err := GetRequesterConfig(cfg, isRequesterNode)
	if err != nil {
		return errors.Wrapf(err, "failed to configure requester node")
	}

	// Create node config from cmd arguments
	hostAddress, err := parseServerAPIHost(cfg.API.Host)
	if err != nil {
		return err
	}
	nodeConfig := node.NodeConfig{
		NodeID:         nodeName,
		CleanupManager: cm,
		DisabledFeatures: node.FeatureConfig{
			Engines:    cfg.Engines.Disabled,
			Publishers: cfg.Publishers.Disabled,
			Storages:   cfg.InputSources.Disabled,
		},
		HostAddress:           hostAddress,
		APIPort:               uint16(cfg.API.Port),
		ComputeConfig:         computeConfig,
		RequesterNodeConfig:   requesterConfig,
		AuthConfig:            cfg.API.Auth,
		IsComputeNode:         isComputeNode,
		IsRequesterNode:       isRequesterNode,
		RequesterSelfSign:     cfg.API.TLS.SelfSigned,
		Labels:                cfg.Compute.Labels,
		AllowListedLocalPaths: cfg.Compute.AllowListedLocalPaths,
		NetworkConfig:         networkConfig,
	}
	if isRequesterNode {
		// We only want auto TLS for the requester node, but this info doesn't fit well
		// with the other data in the requesterConfig.
		nodeConfig.RequesterAutoCert = cfg.API.TLS.AutoCert
		nodeConfig.RequesterAutoCertCache = cfg.API.TLS.AutoCertCachePath
		// If there are configuration values for autocert we should return and let autocert
		// do what it does later on in the setup.
		if nodeConfig.RequesterAutoCert == "" {
			cert, key, err := GetTLSCertificate(ctx, cfg, &nodeConfig)
			if err != nil {
				return err
			}
			nodeConfig.RequesterTLSCertificateFile = cert
			nodeConfig.RequesterTLSKeyFile = key
		}
	}
	// Create node
	log.Info().Msg("starting bacalhau...")
	standardNode, err := node.NewNode(ctx, cfg, nodeConfig, fsRepo)
	if err != nil {
		return fmt.Errorf("error creating node: %w", err)
	}

	// Start node
	if err := standardNode.Start(ctx); err != nil {
		return fmt.Errorf("error starting node: %w", err)
	}

	// Start up Dashboard - default: 8483
	if cfg.WebUI.Enabled {
		apiURL := standardNode.APIServer.GetURI().JoinPath("api", "v1")
		host, portStr, err := net.SplitHostPort(cfg.WebUI.Listen)
		if err != nil {
			return err
		}
		webuiPort, err := strconv.ParseInt(portStr, 10, 64)
		if err != nil {
			return err
		}
		go func() {
			// Specifically leave the host blank. The app will just use whatever
			// host it is served on and replace the port and path.
			apiPort := apiURL.Port()
			apiPath := apiURL.Path

			err := webui.ListenAndServe(ctx, host, apiPort, apiPath, int(webuiPort))
			if err != nil {
				cmd.PrintErrln(err)
			}
		}()
	}

	startupLog := log.Info().
		Str("name", nodeName).
		Str("address", fmt.Sprintf("%s:%d", hostAddress, cfg.API.Port)).
		Bool("compute_enabled", cfg.Compute.Enabled).
		Bool("orchestrator_enabled", cfg.Orchestrator.Enabled).
		Bool("webui_enabled", cfg.WebUI.Enabled)
	if cfg.Compute.Enabled {
		capacity := standardNode.ComputeNode.Capacity.GetMaxCapacity(ctx)
		startupLog.
			Strs("engines", standardNode.ComputeNode.Executors.Keys(ctx)).
			Strs("publishers", standardNode.ComputeNode.Publishers.Keys(ctx)).
			Strs("storages", standardNode.ComputeNode.Storages.Keys(ctx)).
			Strs("orchestrators", cfg.Compute.Orchestrators).
			Str("capacity", capacity.String())

		if len(cfg.Compute.AllowListedLocalPaths) > 0 {
			startupLog.Strs("volumes", cfg.Compute.AllowListedLocalPaths)
		}

	}
	if cfg.Orchestrator.Enabled {
		startupLog.Str("orchestrator_address",
			fmt.Sprintf("%s:%d", cfg.Orchestrator.Host, cfg.Orchestrator.Port))
	}
	startupLog.Msg("bacalhau node running")

	envvars := buildEnvVariables(cfg)
	cmd.Println()
	cmd.Println("To connect to this node from the local client, run the following commands in your shell:")
	cmd.Println(envvars)

	riPath, err := fsRepo.WriteRunInfo(ctx, envvars)
	if err != nil {
		return err
	}
	cmd.Println()
	cmd.Println()
	cmd.Printf("A copy of these variables have been written to: %s\n", riPath)
	defer os.Remove(riPath)

	<-ctx.Done() // block until killed
	log.Info().Msg("bacalhau node shutting down...")
	return nil
}

func GetTLSCertificate(ctx context.Context, cfg types.Bacalhau, nodeConfig *node.NodeConfig) (string, string, error) {
	cert := cfg.API.TLS.CertFile
	key := cfg.API.TLS.KeyFile
	if cert != "" && key != "" {
		return cert, key, nil
	}
	if cert != "" && key == "" {
		return "", "", fmt.Errorf("invalid config: TLS cert specified without corresponding private key")
	}
	if cert == "" && key != "" {
		return "", "", fmt.Errorf("invalid config: private key specified without corresponding TLS certificate")
	}
	if !nodeConfig.RequesterSelfSign {
		return "", "", nil
	}
	log.Ctx(ctx).Info().Msg("Generating self-signed certificate")
	var err error
	// If the user has not specified a private key, use their client key
	if key == "" {
		key, err = cfg.UserKeyPath()
		if err != nil {
			return "", "", err
		}
	}
	certFile, err := os.CreateTemp(os.TempDir(), "bacalhau_cert_*.crt")
	if err != nil {
		return "", "", errors.Wrap(err, "unable to create temporary server certificate")
	}
	defer closer.CloseWithLogOnError(certFile.Name(), certFile)

	var ips []net.IP = nil
	if ip := net.ParseIP(nodeConfig.HostAddress); ip != nil {
		ips = append(ips, ip)
	}

	if privKey, err := crypto.LoadPKCS1KeyFile(key); err != nil {
		return "", "", err
	} else if caCert, err := crypto.NewSelfSignedCertificate(privKey, false, ips); err != nil {
		return "", "", errors.Wrap(err, "failed to generate server certificate")
	} else if err = caCert.MarshalCertficate(certFile); err != nil {
		return "", "", errors.Wrap(err, "failed to write server certificate")
	}
	cert = certFile.Name()
	return cert, key, nil
}

func parseServerAPIHost(host string) (string, error) {
	if net.ParseIP(host) == nil {
		// We should check that the value gives us an address type
		// we can use to get our IP address. If it doesn't, we should
		// panic.
		atype, ok := network.AddressTypeFromString(host)
		if !ok {
			return "", fmt.Errorf("invalid address type in Server API Host config: %s", host)
		}

		addr, err := network.GetNetworkAddress(atype, network.AllAddresses)
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
	if cfg.Orchestrator.Enabled {
		envvars.WriteString(fmt.Sprintf("export %s=%s\n",
			config.KeyAsEnvVar(types.ComputeOrchestratorsKey), getPublicNATSOrchestratorURL(cfg.Orchestrator)))
	}
	return envvars.String()
}

func getAPIURL(cfg types.API) string {
	if cfg.Host == "0.0.0.0" {
		return "127.0.0.1"
	} else {
		return cfg.Host
	}
}

func getPublicNATSOrchestratorURL(cfg types.Orchestrator) *url.URL {
	orchestrator := &url.URL{
		Scheme: "nats",
		Host:   cfg.Advertise,
	}

	if cfg.Advertise == "" {
		orchestrator.Host = fmt.Sprintf("127.0.0.1:%d", cfg.Port)
	}

	return orchestrator
}

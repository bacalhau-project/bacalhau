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
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
	"github.com/bacalhau-project/bacalhau/pkg/lib/crypto"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
	"github.com/bacalhau-project/bacalhau/webui"
)

var (
	serveLong = templates.LongDesc(i18n.T(`
		Start a bacalhau node.
		`))

	serveExample = templates.Examples(i18n.T(`
		# Start a private bacalhau requester node
		bacalhau serve
		# or
		bacalhau serve --orchestrator

		# Start a private bacalhau hybrid node that acts as both compute and requester
		bacalhau serve --orchestrator --compute
		# or

		# Start a public bacalhau node with the WebUI on port 3000 (default:8483)
		bacalhau serve --web-ui --web-ui-port=3000
`))
)

func NewCmd() *cobra.Command {
	serveFlags := map[string][]configflags.Definition{
		"local_publisher":  configflags.LocalPublisherFlags,
		"requester-tls":    configflags.RequesterTLSFlags,
		"server-api":       configflags.ServerAPIFlags,
		"orchestrator":     configflags.OrchestratorFlags,
		"capacity":         configflags.CapacityFlags,
		"ipfs":             configflags.IPFSFlags,
		"compute":          configflags.ComputeFlags,
		"disable-features": configflags.DisabledFeatureFlags,
		"job-selection":    configflags.JobSelectionFlags,
		"labels":           configflags.LabelFlags,
		"web-ui":           configflags.WebUIFlags,
		"node-name":        configflags.NodeNameFlags,
		"translations":     configflags.JobTranslationFlags,
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
func serve(cmd *cobra.Command, cfg types2.Bacalhau, fsRepo *repo.FsRepo) error {
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

	networkConfig, err := getNetworkConfig(cfg)
	if err != nil {
		return err
	}

	// TODO(review) should we build this type iff isComputeNodei == true?
	computeConfig, err := GetComputeConfig(ctx, cfg, isComputeNode)
	if err != nil {
		return errors.Wrapf(err, "failed to configure compute node")
	}

	requesterConfig, err := GetRequesterConfig(cfg, isRequesterNode)
	if err != nil {
		return errors.Wrapf(err, "failed to configure requester node")
	}

	nodeConfig := node.NodeConfig{
		NodeID:         nodeName,
		CleanupManager: cm,
		DisabledFeatures: node.FeatureConfig{
			Engines:    cfg.Engines.Disabled,
			Publishers: cfg.Publishers.Disabled,
			Storages:   cfg.InputSources.Disabled,
		},
		HostAddress:         cfg.API.Host,
		APIPort:             uint16(cfg.API.Port),
		ComputeConfig:       computeConfig,
		RequesterNodeConfig: requesterConfig,
		AuthConfig:          cfg.API.Auth,
		IsComputeNode:       isComputeNode,
		IsRequesterNode:     isRequesterNode,
		Labels:              cfg.Compute.Labels,
		NetworkConfig:       networkConfig,
		RequesterSelfSign:   cfg.API.TLS.SelfSigned,
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

	connectCmd, err := buildConnectCommand(ctx, &nodeConfig)
	if err != nil {
		return err
	}
	cmd.Println()
	cmd.Println(connectCmd)

	cmd.Println()
	cmd.Println("To connect to this node from the client, run the following commands in your shell:")

	<-ctx.Done() // block until killed
	return nil
}

func buildConnectCommand(ctx context.Context, nodeConfig *node.NodeConfig) (string, error) {
	headerB := strings.Builder{}
	cmdB := strings.Builder{}
	if nodeConfig.IsRequesterNode {
		cmdB.WriteString(fmt.Sprintf("%s serve ", os.Args[0]))
		// other nodes can be just compute nodes
		// no need to spawn 1+ requester nodes
		cmdB.WriteString(fmt.Sprintf("--compute "))

		advertisedAddr := getPublicNATSOrchestratorURL(nodeConfig)
		headerB.WriteString("To connect a compute node to this orchestrator, run the following command in your shell:\n")
		cmdB.WriteString(fmt.Sprintf("--orchestrators=%s ", advertisedAddr.String()))
	}

	return headerB.String() + cmdB.String(), nil
}

func getPublicNATSOrchestratorURL(nodeConfig *node.NodeConfig) *url.URL {
	orchestrator := &url.URL{
		Scheme: "nats",
		Host:   nodeConfig.NetworkConfig.AdvertisedAddress,
	}

	if nodeConfig.NetworkConfig.AdvertisedAddress == "" {
		orchestrator.Host = fmt.Sprintf("127.0.0.1:%d", nodeConfig.NetworkConfig.Port)
	}

	return orchestrator
}

func GetTLSCertificate(ctx context.Context, cfg types2.Bacalhau, nodeConfig *node.NodeConfig) (string, string, error) {
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

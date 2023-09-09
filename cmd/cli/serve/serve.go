package serve

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"

	"github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	system_capacity "github.com/bacalhau-project/bacalhau/pkg/compute/capacity/system"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	bac_libp2p "github.com/bacalhau-project/bacalhau/pkg/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"

	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
)

var DefaultSwarmPort = 1235

const DefaultPeerConnect = "none"

var (
	serveLong = templates.LongDesc(i18n.T(`
		Start a bacalhau node.
		`))

	serveExample = templates.Examples(i18n.T(`
		# Start a private bacalhau requester node
		bacalhau serve
		# or
		bacalhau serve --node-type requester

		# Start a private bacalhau hybrid node that acts as both compute and requester
		bacalhau serve --node-type compute --node-type requester
		# or
		bacalhau serve --node-type compute,requester

		# Start a private bacalhau node with a persistent local IPFS node
		BACALHAU_SERVE_IPFS_PATH=/data/ipfs bacalhau serve

		# Start a public bacalhau requester node
		bacalhau serve --peer env --private-internal-ipfs=false
`))
)

func GetPeers(peerConnect string) ([]multiaddr.Multiaddr, error) {
	var (
		peersStrings []string
	)
	// TODO(forrest): [ux] this is a really confusing way to configure bootstrap peers.
	// The convenience is nice by passing a single 'env' value, and can be improved with sane defaults commented
	// out in the config. If a user wants to connect then can pass the --peer flag or uncomment the config values.
	if peerConnect == DefaultPeerConnect || peerConnect == "" {
		return nil, nil
	} else if peerConnect == "env" {
		// TODO(forrest): [ux/sanity] in the future default to the value in the config file and remove system environment
		peersStrings = system.Envs[system.GetEnvironment()].BootstrapAddresses
	} else {
		peersStrings = strings.Split(peerConnect, ",")
	}

	peers := make([]multiaddr.Multiaddr, 0, len(peersStrings))
	for _, peer := range peersStrings {
		parsed, err := multiaddr.NewMultiaddr(peer)
		if err != nil {
			return nil, err
		}
		peers = append(peers, parsed)
	}
	return peers, nil
}

func NewCmd() *cobra.Command {
	serveFlags := map[string][]configflags.Definition{
		"requester-tls":    configflags.RequesterTLSFlags,
		"server-api":       configflags.ServerAPIFlags,
		"libp2p":           configflags.Libp2pFlags,
		"ipfs":             configflags.IPFSFlags,
		"capacity":         configflags.CapacityFlags,
		"job-selection":    configflags.JobSelectionFlags,
		"disable-features": configflags.DisabledFeatureFlags,
		"labels":           configflags.LabelFlags,
		"node-type":        configflags.NodeTypeFlags,
		"list-local":       configflags.AllowListLocalPathsFlags,
		"compute-store":    configflags.ComputeStorageFlags,
		"requester-store":  configflags.RequesterJobStorageFlags,
	}

	serveCmd := &cobra.Command{
		Use:     "serve",
		Short:   "Start the bacalhau compute node",
		Long:    serveLong,
		Example: serveExample,
		PreRun: func(cmd *cobra.Command, args []string) {
			/*
				NB(forrest):
				(I learned a lot more about viper and cobra than was intended...)

				Binding flags in the PreRun phase is crucial to ensure that Viper binds only
				to the flags specific to the current command being executed. This helps prevent
				potential issues with overlapping flag names or default values from other commands.
				An example of an overlapping flagset is the libp2p flags, shared here and in the id command.
				By binding in PreRun, we maintain a clean separation between flag registration
				and its binding to configuration. It ensures that each command's flags are
				independently managed, avoiding interference or unexpected behavior from shared
				flag names across multiple commands.

				It's essential to understand the nature of Viper when working with Cobra commands.
				At its core, Viper functions as a flat namespace key-value store for configuration
				settings. It doesn't inherently respect or understand Cobra's command hierarchy or
				nuances. When multiple commands have overlapping flag names and modify the same
				configuration key in Viper, there's potential for confusion. For example, if two
				commands both use a "peer" flag, which value should Viper return and how does it
				know if the flag changes? Since Viper doesn't recognize the context of commands, it will
				return the value of the last flag bound to it. This is why it's important to manage
				flag binding thoughtfully, ensuring each command's context is respected.
			*/
			if err := configflags.BindFlags(cmd, serveFlags); err != nil {
				util.Fatal(cmd, err, 1)
			}
		},
		Run: func(cmd *cobra.Command, _ []string) {
			if err := serve(cmd); err != nil {
				util.Fatal(cmd, err, 1)
			}
		},
	}

	if err := configflags.RegisterFlags(serveCmd, serveFlags); err != nil {
		util.Fatal(serveCmd, err, 1)
	}

	return serveCmd
}

//nolint:funlen,gocyclo
func serve(cmd *cobra.Command) error {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	ctx := cmd.Context()
	cm := util.GetCleanupManager(ctx)

	// load the repo and its config file, reading in the values, flags and env vars will override values in config.
	fsRepo, err := repo.NewFS(viper.GetString("repo"))
	if err != nil {
		return err
	}
	if err := fsRepo.Open(); err != nil {
		return err
	}

	// configure node type
	isRequesterNode, isComputeNode, err := getNodeType()
	if err != nil {
		return err
	}

	libp2pCfg, err := config.GetLibp2pConfig()
	if err != nil {
		return err
	}

	peers, err := GetPeers(libp2pCfg.PeerConnect)
	if err != nil {
		return err
	}

	// configure libp2p
	libp2pHost, err := SetupLibp2pHost(libp2pCfg)
	if err != nil {
		return err
	}
	cm.RegisterCallback(libp2pHost.Close)
	// add nodeID to logging context
	ctx = logger.ContextWithNodeIDLogger(ctx, libp2pHost.ID().String())

	// Establishing IPFS connection
	ipfsConfig, err := getIPFSConfig()
	if err != nil {
		return err
	}

	ipfsClient, err := SetupIPFSClient(ctx, cm, ipfsConfig)
	if err != nil {
		return err
	}

	computeConfig, err := GetComputeConfig()
	if err != nil {
		return err
	}

	requesterConfig, err := GetRequesterConfig()
	if err != nil {
		return err
	}

	featureConfig, err := getDisabledFeatures()
	if err != nil {
		return err
	}

	allowedListLocalPaths := getAllowListedLocalPathsConfig()

	// TODO (forrest): [ux] in the future we should make this configurable to users.
	autoLabel := true
	// Create node config from cmd arguments
	nodeConfig := node.NodeConfig{
		CleanupManager:        cm,
		IPFSClient:            ipfsClient,
		Host:                  libp2pHost,
		DisabledFeatures:      featureConfig,
		HostAddress:           config.ServerAPIHost(),
		APIPort:               config.ServerAPIPort(),
		ComputeConfig:         computeConfig,
		RequesterNodeConfig:   requesterConfig,
		IsComputeNode:         isComputeNode,
		IsRequesterNode:       isRequesterNode,
		Labels:                getNodeLabels(autoLabel),
		AllowListedLocalPaths: allowedListLocalPaths,
		FsRepo:                fsRepo,
	}

	if isRequesterNode {
		// We only want auto TLS for the requester node, but this info doesn't fit well
		// with the other data in the requesterConfig.
		nodeConfig.RequesterAutoCert = config.ServerAutoCertDomain()
		nodeConfig.RequesterAutoCertCache = config.GetAutoCertCachePath()
	}

	stopFn, err := node.NewNodeWithOptions(ctx, nodeConfig)
	if err != nil {
		panic(err)
	}

	// Create node
	/*
		standardNode, err := node.NewNode(ctx, nodeConfig)
		if err != nil {
			return fmt.Errorf("error creating node: %w", err)
		}
	*/

	// Start transport layer
	err = bac_libp2p.ConnectToPeersContinuously(ctx, cm, libp2pHost, peers)
	if err != nil {
		return err
	}

	/*
		// Start node
		if err := standardNode.Start(ctx); err != nil {
			return fmt.Errorf("error starting node: %w", err)
		}

		// only in station logging output
		if config.GetLogMode() == logger.LogModeStation && standardNode.IsComputeNode() {
			cmd.Printf("API: %s\n", standardNode.APIServer.GetURI().JoinPath("/api/v1/compute/debug"))
		}

	*/

	if ipfsConfig.PrivateInternal && libp2pCfg.PeerConnect == DefaultPeerConnect {
		// other nodes can be just compute nodes
		// no need to spawn 1+ requester nodes
		nodeType := "--node-type compute"

		if isComputeNode && !isRequesterNode {
			cmd.Println("Make sure there's at least one requester node in your network.")
		}

		ipfsAddresses, err := ipfsClient.SwarmMultiAddresses(ctx)
		if err != nil {
			return fmt.Errorf("error looking up IPFS addresses: %w", err)
		}

		p2pAddr, err := multiaddr.NewMultiaddr("/p2p/" + libp2pHost.ID().String())
		if err != nil {
			return err
		}

		peerAddress := pickP2pAddress(libp2pHost.Addrs()).Encapsulate(p2pAddr).String()
		ipfsSwarmAddress := pickP2pAddress(ipfsAddresses).String()

		cmd.Println()
		cmd.Println("To connect another node to this private one, run the following command in your shell:")
		cmd.Printf(
			"%s serve %s --private-internal-ipfs --peer %s --ipfs-swarm-addr %s\n",
			os.Args[0], nodeType, peerAddress, ipfsSwarmAddress,
		)

		summaryBuilder := strings.Builder{}
		summaryBuilder.WriteString(fmt.Sprintf(
			"export %s=%s\n",
			config.KeyAsEnvVar(types.NodeIPFSSwarmAddresses),
			ipfsSwarmAddress,
		))
		summaryBuilder.WriteString(fmt.Sprintf(
			"export %s=%s\n",
			config.KeyAsEnvVar(types.NodeClientAPIHost),
			config.ServerAPIHost(),
		))
		summaryBuilder.WriteString(fmt.Sprintf(
			"export %s=%d\n",
			config.KeyAsEnvVar(types.NodeClientAPIPort),
			config.ServerAPIPort(),
		))
		summaryBuilder.WriteString(fmt.Sprintf(
			"export %s=%s\n",
			config.KeyAsEnvVar(types.NodeLibp2pPeerConnect),
			peerAddress,
		))

		// Just convenience below - print out the last of the nodes information as the global variable
		summaryShellVariablesString := summaryBuilder.String()

		if isRequesterNode {
			cmd.Println()
			cmd.Println("To use this requester node from the client, run the following commands in your shell:")
			cmd.Println(summaryShellVariablesString)
		}

		ripath, err := fsRepo.WriteRunInfo(ctx, summaryShellVariablesString)
		if err != nil {
			return fmt.Errorf("writing run info to repo: %w", err)
		} else {
			cmd.Printf("A copy of these variables have been written to: %s\n", ripath)
		}
		if err != nil {
			return err
		}

		cm.RegisterCallback(func() error {
			return os.Remove(ripath)
		})
	}

	<-ctx.Done() // block until killed
	return stopFn(context.TODO())
}

// pickP2pAddress will aim to select a non-localhost IPv4 TCP address, or at least a non-localhost IPv6 one, from a list
// of addresses.
func pickP2pAddress(addresses []multiaddr.Multiaddr) multiaddr.Multiaddr {
	value := func(m multiaddr.Multiaddr) int {
		count := 0
		if _, err := m.ValueForProtocol(multiaddr.P_TCP); err == nil {
			count++
		}
		if ip, err := m.ValueForProtocol(multiaddr.P_IP4); err == nil {
			count++
			if ip != "127.0.0.1" {
				count++
			}
		} else if ip, err := m.ValueForProtocol(multiaddr.P_IP6); err == nil && ip != "::1" {
			count++
		}
		return count
	}

	preferredAddress := config.PreferredAddress()
	if preferredAddress != "" {
		for _, addr := range addresses {
			if strings.Contains(addr.String(), preferredAddress) {
				return addr
			}
		}
	}

	sort.Slice(addresses, func(i, j int) bool {
		return value(addresses[i]) > value(addresses[j])
	})

	return addresses[0]
}

func AutoOutputLabels() map[string]string {
	m := make(map[string]string)
	// Get the operating system name
	os := runtime.GOOS
	m["Operating-System"] = os
	m["git-lfs"] = "False"
	if checkGitLFS() {
		m["git-lfs"] = "True"
	}
	arch := runtime.GOARCH
	m["Architecture"] = arch

	gpus, err := system_capacity.GetSystemGPUs()
	if err != nil {
		// Print the GPU names
		for i, gpu := range gpus {
			// Model label e.g. GPU-0: Tesla-T1
			key := fmt.Sprintf("GPU-%d", gpu.Index)
			name := strings.Replace(gpu.Name, " ", "-", -1) // Replace spaces with dashes
			m[key] = name

			// Memory label e.g. GPU-0-Memory: 15360-MiB
			key = fmt.Sprintf("GPU-%d-Memory", i)
			memory := strings.Replace(fmt.Sprintf("%d MiB", gpu.Memory), " ", "-", -1) // Replace spaces with dashes
			m[key] = memory
		}
	}
	// Get list of installed packages (Only works for linux, make it work for every platform)
	// files, err := ioutil.ReadDir("/var/lib/dpkg/info")
	// if err != nil {
	// 	panic(err)
	// }
	// var packageList []string
	// for _, file := range files {
	// 	if !file.IsDir() && filepath.Ext(file.FlagName()) == ".list" {

	// 		packageList = append(packageList, file.FlagName()[:len(file.FlagName())-5])
	// 	}
	// }
	// m["Installed-Packages"] = strings.Join(packageList, ",")
	return m
}

func checkGitLFS() bool {
	_, err := exec.LookPath("git-lfs")
	return err == nil
}

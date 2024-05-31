package serve

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	libp2p_crypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/multiformats/go-multiaddr"
	pkgerrors "github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	boltjobstore "github.com/bacalhau-project/bacalhau/pkg/jobstore/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/lib/network"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/transformer"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func GetComputeConfig(ctx context.Context, cfg types.NodeConfig, createExecutionStore bool) (node.ComputeConfig, error) {
	totalResources, totalErr := cfg.Compute.Capacity.TotalResourceLimits.ToResources()
	jobResources, jobErr := cfg.Compute.Capacity.JobResourceLimits.ToResources()
	defaultResources, defaultErr := cfg.Compute.Capacity.DefaultJobResourceLimits.ToResources()
	if err := errors.Join(totalErr, jobErr, defaultErr); err != nil {
		return node.ComputeConfig{}, err
	}

	var err error
	var executionStore store.ExecutionStore

	if createExecutionStore {
		executionStore, err = getExecutionStore(ctx, cfg.Compute.ExecutionStore)
		if err != nil {
			return node.ComputeConfig{}, pkgerrors.Wrapf(err, "failed to create execution store")
		}
	}

	return node.NewComputeConfigWith(cfg.ComputeStoragePath, node.ComputeConfigParams{
		TotalResourceLimits:                   *totalResources,
		JobResourceLimits:                     *jobResources,
		DefaultJobResourceLimits:              *defaultResources,
		IgnorePhysicalResourceLimits:          cfg.Compute.Capacity.IgnorePhysicalResourceLimits,
		JobNegotiationTimeout:                 time.Duration(cfg.Compute.JobTimeouts.JobNegotiationTimeout),
		MinJobExecutionTimeout:                time.Duration(cfg.Compute.JobTimeouts.MinJobExecutionTimeout),
		MaxJobExecutionTimeout:                time.Duration(cfg.Compute.JobTimeouts.MaxJobExecutionTimeout),
		DefaultJobExecutionTimeout:            time.Duration(cfg.Compute.JobTimeouts.DefaultJobExecutionTimeout),
		JobExecutionTimeoutClientIDBypassList: cfg.Compute.JobTimeouts.JobExecutionTimeoutClientIDBypassList,
		JobSelectionPolicy: node.JobSelectionPolicy{
			Locality:            semantic.JobSelectionDataLocality(cfg.Compute.JobSelection.Locality),
			RejectStatelessJobs: cfg.Compute.JobSelection.RejectStatelessJobs,
			AcceptNetworkedJobs: cfg.Compute.JobSelection.AcceptNetworkedJobs,
			ProbeHTTP:           cfg.Compute.JobSelection.ProbeHTTP,
			ProbeExec:           cfg.Compute.JobSelection.ProbeExec,
		},
		LogRunningExecutionsInterval: time.Duration(cfg.Compute.Logging.LogRunningExecutionsInterval),
		LogStreamBufferSize:          cfg.Compute.LogStreamConfig.ChannelBufferSize,
		ExecutionStore:               executionStore,
		LocalPublisher:               cfg.Compute.LocalPublisher,
	})
}

func GetRequesterConfig(ctx context.Context, cfg types.RequesterConfig, createJobStore bool) (node.RequesterConfig, error) {
	var err error
	var jobStore jobstore.Store
	if createJobStore {
		jobStore, err = getJobStore(ctx, cfg.JobStore)
		if err != nil {
			return node.RequesterConfig{}, pkgerrors.Wrapf(err, "failed to create job store")
		}
	}

	requesterConfig, err := node.NewRequesterConfigWith(node.RequesterConfigParams{
		JobDefaults: transformer.JobDefaults{
			ExecutionTimeout: time.Duration(cfg.JobDefaults.ExecutionTimeout),
		},
		HousekeepingBackgroundTaskInterval: time.Duration(cfg.HousekeepingBackgroundTaskInterval),
		NodeRankRandomnessRange:            cfg.NodeRankRandomnessRange,
		OverAskForBidsFactor:               cfg.OverAskForBidsFactor,
		JobSelectionPolicy: node.JobSelectionPolicy{
			Locality:            semantic.JobSelectionDataLocality(cfg.JobSelectionPolicy.Locality),
			RejectStatelessJobs: cfg.JobSelectionPolicy.RejectStatelessJobs,
			AcceptNetworkedJobs: cfg.JobSelectionPolicy.AcceptNetworkedJobs,
			ProbeHTTP:           cfg.JobSelectionPolicy.ProbeHTTP,
			ProbeExec:           cfg.JobSelectionPolicy.ProbeExec,
		},
		FailureInjectionConfig:         cfg.FailureInjectionConfig,
		EvalBrokerVisibilityTimeout:    time.Duration(cfg.EvaluationBroker.EvalBrokerVisibilityTimeout),
		EvalBrokerInitialRetryDelay:    time.Duration(cfg.EvaluationBroker.EvalBrokerInitialRetryDelay),
		EvalBrokerSubsequentRetryDelay: time.Duration(cfg.EvaluationBroker.EvalBrokerSubsequentRetryDelay),
		EvalBrokerMaxRetryCount:        cfg.EvaluationBroker.EvalBrokerMaxRetryCount,
		WorkerCount:                    cfg.Worker.WorkerCount,
		WorkerEvalDequeueTimeout:       time.Duration(cfg.Worker.WorkerEvalDequeueTimeout),
		WorkerEvalDequeueBaseBackoff:   time.Duration(cfg.Worker.WorkerEvalDequeueBaseBackoff),
		WorkerEvalDequeueMaxBackoff:    time.Duration(cfg.Worker.WorkerEvalDequeueMaxBackoff),
		S3PreSignedURLExpiration:       time.Duration(cfg.StorageProvider.S3.PreSignedURLExpiration),
		S3PreSignedURLDisabled:         cfg.StorageProvider.S3.PreSignedURLDisabled,
		TranslationEnabled:             cfg.TranslationEnabled,
		JobStore:                       jobStore,
		DefaultPublisher:               cfg.DefaultPublisher,
	})
	if err != nil {
		return node.RequesterConfig{}, err
	}

	if cfg.ManualNodeApproval {
		requesterConfig.DefaultApprovalState = models.NodeMembership.PENDING
	} else {
		requesterConfig.DefaultApprovalState = models.NodeMembership.APPROVED
	}

	return requesterConfig, nil
}

func getNodeType(cfg types.BacalhauConfig) (requester, compute bool, err error) {
	requester = false
	compute = false
	err = nil

	for _, nodeType := range cfg.Node.Type {
		if nodeType == "compute" {
			compute = true
		} else if nodeType == "requester" {
			requester = true
		} else {
			err = fmt.Errorf("invalid node type %s. Only compute and requester values are supported", nodeType)
		}
	}
	return
}

func getIPFSConfig(ipfsConfig types.IpfsConfig) (types.IpfsConfig, error) {
	// TODO this can be moved to a validate method on the IpfsConfig type
	if ipfsConfig.Connect != "" && ipfsConfig.PrivateInternal {
		return types.IpfsConfig{}, fmt.Errorf("%s cannot be used with %s",
			configflags.FlagNameForKey(types.NodeIPFSPrivateInternal, configflags.IPFSFlags...),
			configflags.FlagNameForKey(types.NodeIPFSConnect, configflags.IPFSFlags...),
		)
	}

	return ipfsConfig, nil
}

func SetupIPFSClient(ctx context.Context, cm *system.CleanupManager, ipfsCfg types.IpfsConfig) (ipfs.Client, error) {
	if ipfsCfg.Connect == "" {
		ipfsNode, err := ipfs.NewNodeWithConfig(ctx, cm, ipfsCfg)
		if err != nil {
			return ipfs.Client{}, fmt.Errorf("error creating IPFS node: %s", err)
		}
		if ipfsCfg.PrivateInternal {
			log.Ctx(ctx).Debug().Msgf("ipfs_node_api_port: %d", ipfsNode.APIPort)
		}
		cm.RegisterCallbackWithContext(ipfsNode.Close)
		client := ipfsNode.Client()

		swarmAddresses, err := client.SwarmAddresses(ctx)
		if err != nil {
			return ipfs.Client{}, fmt.Errorf("error looking up IPFS addresses: %s", err)
		}

		log.Ctx(ctx).Debug().Strs("ipfs_swarm_addresses", swarmAddresses).Msg("Internal IPFS node available")
		return client, nil
	}

	client, err := ipfs.NewClientUsingRemoteHandler(ctx, ipfsCfg.Connect)
	if err != nil {
		return ipfs.Client{}, fmt.Errorf("error creating IPFS client: %s", err)
	}

	if len(ipfsCfg.SwarmAddresses) != 0 {
		maddrs, err := ipfs.ParsePeersString(ipfsCfg.SwarmAddresses)
		if err != nil {
			return ipfs.Client{}, err
		}

		client.SwarmConnect(ctx, maddrs)
	}

	return client, nil
}

func getNetworkConfig(cfg types.NetworkConfig) (node.NetworkConfig, error) {
	return node.NetworkConfig{
		Type:                     cfg.Type,
		Port:                     cfg.Port,
		AdvertisedAddress:        cfg.AdvertisedAddress,
		Orchestrators:            cfg.Orchestrators,
		StoreDir:                 cfg.StoreDir,
		AuthSecret:               cfg.AuthSecret,
		ClusterName:              cfg.Cluster.Name,
		ClusterPort:              cfg.Cluster.Port,
		ClusterAdvertisedAddress: cfg.Cluster.AdvertisedAddress,
		ClusterPeers:             cfg.Cluster.Peers,
	}, nil
}

func getExecutionStore(ctx context.Context, storeCfg types.JobStoreConfig) (store.ExecutionStore, error) {
	if err := storeCfg.Validate(); err != nil {
		return nil, err
	}

	switch storeCfg.Type {
	case types.BoltDB:
		return boltdb.NewStore(ctx, storeCfg.Path)
	default:
		return nil, fmt.Errorf("unknown JobStore type: %s", storeCfg.Type)
	}
}

func getJobStore(ctx context.Context, storeCfg types.JobStoreConfig) (jobstore.Store, error) {
	if err := storeCfg.Validate(); err != nil {
		return nil, err
	}

	switch storeCfg.Type {
	case types.BoltDB:
		log.Ctx(ctx).Debug().Str("Path", storeCfg.Path).Msg("creating boltdb backed jobstore")
		return boltjobstore.NewBoltJobStore(storeCfg.Path)
	default:
		return nil, fmt.Errorf("unknown JobStore type: %s", storeCfg.Type)
	}
}

func getNodeID(ctx context.Context, nodeNameProviderType string) (string, error) {
	nodeNameProviders := map[string]idgen.NodeNameProvider{
		"hostname": idgen.HostnameProvider{},
		"aws":      idgen.NewAWSNodeNameProvider(),
		"gcp":      idgen.NewGCPNodeNameProvider(),
		"uuid":     idgen.UUIDNodeNameProvider{},
		"puuid":    idgen.PUUIDNodeNameProvider{},
	}
	nodeNameProvider, ok := nodeNameProviders[nodeNameProviderType]
	if !ok {
		return "", fmt.Errorf(
			"unknown node name provider: %s. Supported providers are: %s", nodeNameProviderType, lo.Keys(nodeNameProviders))
	}

	nodeName, err := nodeNameProvider.GenerateNodeName(ctx)
	if err != nil {
		return "", err
	}

	return nodeName, nil
}

// persistConfigs writes the resolved config to the persisted config file.
// this will only write values that must not change between invocations,
// such as the job store path and node name,
// and only if they are not already set in the config file.
func persistConfigs(repoPath string, cfg types.BacalhauConfig) error {
	if err := config.WritePersistedConfigs(filepath.Join(repoPath, config.FileName), cfg); err != nil {
		return fmt.Errorf("error writing persisted config: %w", err)
	}
	return nil
}

func loadLibp2pPrivKey(keyPath string) (libp2p_crypto.PrivKey, error) {
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}
	// base64 decode keyBytes
	b64, err := base64.StdEncoding.DecodeString(string(keyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}
	// parse the private key
	key, err := libp2p_crypto.UnmarshalPrivateKey(b64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	return key, nil
}

func parseBootstrapPeers(bootstrappers []string) ([]multiaddr.Multiaddr, error) {
	peers := make([]multiaddr.Multiaddr, 0, len(bootstrappers))
	for _, peer := range bootstrappers {
		parsed, err := multiaddr.NewMultiaddr(peer)
		if err != nil {
			return nil, err
		}
		peers = append(peers, parsed)
	}
	return peers, nil
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

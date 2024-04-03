package serve

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"github.com/spf13/viper"
	"go.uber.org/multierr"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	boltjobstore "github.com/bacalhau-project/bacalhau/pkg/jobstore/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"

	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/transformer"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func GetComputeConfig(ctx context.Context, createExecutionStore bool) (node.ComputeConfig, error) {
	var cfg types.ComputeConfig
	if err := config.ForKey(types.NodeCompute, &cfg); err != nil {
		return node.ComputeConfig{}, err
	}

	totalResources, totalErr := cfg.Capacity.TotalResourceLimits.ToResources()
	queueResources, queueErr := cfg.Capacity.QueueResourceLimits.ToResources()
	jobResources, jobErr := cfg.Capacity.JobResourceLimits.ToResources()
	defaultResources, defaultErr := cfg.Capacity.DefaultJobResourceLimits.ToResources()
	if err := multierr.Combine(totalErr, queueErr, jobErr, defaultErr); err != nil {
		return node.ComputeConfig{}, err
	}

	var err error
	var executionStore store.ExecutionStore

	if createExecutionStore {
		executionStore, err = getExecutionStore(ctx, cfg.ExecutionStore)
		if err != nil {
			return node.ComputeConfig{}, errors.Wrapf(err, "failed to create execution store")
		}
	}

	return node.NewComputeConfigWith(node.ComputeConfigParams{
		TotalResourceLimits:                   *totalResources,
		QueueResourceLimits:                   *queueResources,
		JobResourceLimits:                     *jobResources,
		DefaultJobResourceLimits:              *defaultResources,
		IgnorePhysicalResourceLimits:          cfg.Capacity.IgnorePhysicalResourceLimits,
		JobNegotiationTimeout:                 time.Duration(cfg.JobTimeouts.JobNegotiationTimeout),
		MinJobExecutionTimeout:                time.Duration(cfg.JobTimeouts.MinJobExecutionTimeout),
		MaxJobExecutionTimeout:                time.Duration(cfg.JobTimeouts.MaxJobExecutionTimeout),
		DefaultJobExecutionTimeout:            time.Duration(cfg.JobTimeouts.DefaultJobExecutionTimeout),
		JobExecutionTimeoutClientIDBypassList: cfg.JobTimeouts.JobExecutionTimeoutClientIDBypassList,
		JobSelectionPolicy: node.JobSelectionPolicy{
			Locality:            semantic.JobSelectionDataLocality(cfg.JobSelection.Locality),
			RejectStatelessJobs: cfg.JobSelection.RejectStatelessJobs,
			AcceptNetworkedJobs: cfg.JobSelection.AcceptNetworkedJobs,
			ProbeHTTP:           cfg.JobSelection.ProbeHTTP,
			ProbeExec:           cfg.JobSelection.ProbeExec,
		},
		LogRunningExecutionsInterval: time.Duration(cfg.Logging.LogRunningExecutionsInterval),
		LogStreamBufferSize:          cfg.LogStreamConfig.ChannelBufferSize,
		ExecutionStore:               executionStore,
		LocalPublisher:               cfg.LocalPublisher,
	})
}

func GetRequesterConfig(ctx context.Context, createJobStore bool) (node.RequesterConfig, error) {
	var cfg types.RequesterConfig
	if err := config.ForKey(types.NodeRequester, &cfg); err != nil {
		return node.RequesterConfig{}, err
	}

	var err error
	var jobStore jobstore.Store
	if createJobStore {
		jobStore, err = getJobStore(ctx, cfg.JobStore)
		if err != nil {
			return node.RequesterConfig{}, errors.Wrapf(err, "failed to create job store")
		}
	}
	return node.NewRequesterConfigWith(node.RequesterConfigParams{
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
}

func getNodeType() (requester, compute bool, err error) {
	requester = false
	compute = false
	err = nil

	nodeType := viper.GetStringSlice(types.NodeType)
	for _, nodeType := range nodeType {
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

func getIPFSConfig() (types.IpfsConfig, error) {
	var ipfsConfig types.IpfsConfig
	if err := config.ForKey(types.NodeIPFS, &ipfsConfig); err != nil {
		return types.IpfsConfig{}, err
	}

	if ipfsConfig.Connect != "" && ipfsConfig.PrivateInternal {
		ipfsConfig.PrivateInternal = false
		log.Debug().Msg("disabling ipfs private internal")
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

func getAllowListedLocalPathsConfig() []string {
	return viper.GetStringSlice(types.NodeAllowListedLocalPaths)
}

func getTransportType() (string, error) {
	var networkCfg types.NetworkConfig
	if err := config.ForKey(types.NodeNetwork, &networkCfg); err != nil {
		return "", err
	}
	return networkCfg.Type, nil
}

func getNetworkConfig() (node.NetworkConfig, error) {
	var networkCfg types.NetworkConfig
	if err := config.ForKey(types.NodeNetwork, &networkCfg); err != nil {
		return node.NetworkConfig{}, err
	}

	return node.NetworkConfig{
		Type:                     networkCfg.Type,
		Port:                     networkCfg.Port,
		AdvertisedAddress:        networkCfg.AdvertisedAddress,
		Orchestrators:            networkCfg.Orchestrators,
		StoreDir:                 networkCfg.StoreDir,
		AuthSecret:               networkCfg.AuthSecret,
		ClusterName:              networkCfg.Cluster.Name,
		ClusterPort:              networkCfg.Cluster.Port,
		ClusterAdvertisedAddress: networkCfg.Cluster.AdvertisedAddress,
		ClusterPeers:             networkCfg.Cluster.Peers,
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

func getNodeID(ctx context.Context) (string, error) {
	nodeName, err := config.Get[string](types.NodeName)
	if err != nil {
		return "", err
	}

	if nodeName != "" {
		return nodeName, nil
	}
	nodeNameProviderType, err := config.Get[string](types.NodeNameProvider)
	if err != nil {
		return "", err
	}

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

	nodeName, err = nodeNameProvider.GenerateNodeName(ctx)
	if err != nil {
		return "", err
	}

	// set the new name in the config, so it can be used and persisted later.
	config.SetValue(types.NodeName, nodeName)
	return nodeName, nil
}

// persistConfigs writes the resolved config to the persisted config file.
// this will only write values that must not change between invocations,
// such as the job store path and node name,
// and only if they are not already set in the config file.
func persistConfigs(repoPath string) error {
	resolvedConfig, err := config.GetConfig()
	if err != nil {
		return fmt.Errorf("error getting config: %w", err)
	}
	err = config.WritePersistedConfigs(filepath.Join(repoPath, config.ConfigFileName), *resolvedConfig)
	if err != nil {
		return fmt.Errorf("error writing persisted config: %w", err)
	}
	return nil
}

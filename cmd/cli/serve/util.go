package serve

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"go.uber.org/multierr"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/transformer"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func GetComputeConfig() (node.ComputeConfig, error) {
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
	})
}

func GetRequesterConfig() (node.RequesterConfig, error) {
	var cfg types.RequesterConfig
	if err := config.ForKey(types.NodeRequester, &cfg); err != nil {
		return node.RequesterConfig{}, err
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

func getAllowListedLocalPathsConfig() []string {
	return viper.GetStringSlice(types.NodeAllowListedLocalPaths)
}

func getNetworkConfig(nodeID string) (node.NetworkConfig, error) {
	var networkCfg types.NetworkConfig
	if err := config.ForKey(types.NodeNetwork, &networkCfg); err != nil {
		return node.NetworkConfig{}, err
	}

	// Generate an auth token using the user's client key. It should not be
	// possible to compute this value by anyone other than the NATS server, and
	// should be stable across restarts of the node.
	var keySig string
	keySig, err := system.SignForClient([]byte(nodeID))
	if err != nil {
		return node.NetworkConfig{}, err
	}
	hash := sha256.Sum256([]byte(keySig))
	token := base64.RawURLEncoding.EncodeToString(hash[:])

	return node.NetworkConfig{
		Type:                     networkCfg.Type,
		Port:                     networkCfg.Port,
		AdvertisedAddress:        networkCfg.AdvertisedAddress,
		Orchestrators:            networkCfg.Orchestrators,
		AuthSecret:               token,
		ClusterName:              networkCfg.Cluster.Name,
		ClusterPort:              networkCfg.Cluster.Port,
		ClusterAdvertisedAddress: networkCfg.Cluster.AdvertisedAddress,
		ClusterPeers:             networkCfg.Cluster.Peers,
	}, nil
}

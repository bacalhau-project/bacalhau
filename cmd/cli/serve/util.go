package serve

import (
	"context"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"time"

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

	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/transformer"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/node"
)

func GetComputeConfig(
	ctx context.Context,
	cfg types.BacalhauConfig,
	createExecutionStore bool,
) (node.ComputeConfig, error) {
	totalResources, totalErr := cfg.Node.Compute.Capacity.TotalResourceLimits.ToResources()
	jobResources, jobErr := cfg.Node.Compute.Capacity.JobResourceLimits.ToResources()
	defaultResources, defaultErr := cfg.Node.Compute.Capacity.DefaultJobResourceLimits.ToResources()
	if err := errors.Join(totalErr, jobErr, defaultErr); err != nil {
		return node.ComputeConfig{}, err
	}

	var executionStore store.ExecutionStore
	if createExecutionStore {
		var err error
		executionStore, err = getExecutionStore(ctx, cfg.Node.Compute.ExecutionStore, cfg.ExecutionStorePath())
		if err != nil {
			return node.ComputeConfig{}, pkgerrors.Wrapf(err, "failed to create execution store")
		}
	}

	return node.NewComputeConfigWith(cfg.ExecutionDir(), node.ComputeConfigParams{
		TotalResourceLimits:                   *totalResources,
		JobResourceLimits:                     *jobResources,
		DefaultJobResourceLimits:              *defaultResources,
		IgnorePhysicalResourceLimits:          cfg.Node.Compute.Capacity.IgnorePhysicalResourceLimits,
		JobNegotiationTimeout:                 time.Duration(cfg.Node.Compute.JobTimeouts.JobNegotiationTimeout),
		MinJobExecutionTimeout:                time.Duration(cfg.Node.Compute.JobTimeouts.MinJobExecutionTimeout),
		MaxJobExecutionTimeout:                time.Duration(cfg.Node.Compute.JobTimeouts.MaxJobExecutionTimeout),
		DefaultJobExecutionTimeout:            time.Duration(cfg.Node.Compute.JobTimeouts.DefaultJobExecutionTimeout),
		JobExecutionTimeoutClientIDBypassList: cfg.Node.Compute.JobTimeouts.JobExecutionTimeoutClientIDBypassList,
		JobSelectionPolicy: node.JobSelectionPolicy{
			Locality:            semantic.JobSelectionDataLocality(cfg.Node.Compute.JobSelection.Locality),
			RejectStatelessJobs: cfg.Node.Compute.JobSelection.RejectStatelessJobs,
			AcceptNetworkedJobs: cfg.Node.Compute.JobSelection.AcceptNetworkedJobs,
			ProbeHTTP:           cfg.Node.Compute.JobSelection.ProbeHTTP,
			ProbeExec:           cfg.Node.Compute.JobSelection.ProbeExec,
		},
		LogRunningExecutionsInterval: time.Duration(cfg.Node.Compute.Logging.LogRunningExecutionsInterval),
		LogStreamBufferSize:          cfg.Node.Compute.LogStreamConfig.ChannelBufferSize,
		ExecutionStore:               executionStore,
		LocalPublisher:               cfg.Node.Compute.LocalPublisher,
	})
}

func GetRequesterConfig(ctx context.Context, cfg types.BacalhauConfig, createJobStore bool) (node.RequesterConfig, error) {
	var err error
	var jobStore jobstore.Store
	if createJobStore {
		jobStore, err = getJobStore(ctx, cfg.Node.Requester.JobStore, cfg.JobStorePath())
		if err != nil {
			return node.RequesterConfig{}, pkgerrors.Wrapf(err, "failed to create job store")
		}
	}

	requesterConfig, err := node.NewRequesterConfigWith(node.RequesterConfigParams{
		JobDefaults: transformer.JobDefaults{
			TotalTimeout:     time.Duration(cfg.Node.Requester.JobDefaults.TotalTimeout),
			ExecutionTimeout: time.Duration(cfg.Node.Requester.JobDefaults.ExecutionTimeout),
			QueueTimeout:     time.Duration(cfg.Node.Requester.JobDefaults.QueueTimeout),
		},
		HousekeepingBackgroundTaskInterval: time.Duration(cfg.Node.Requester.HousekeepingBackgroundTaskInterval),
		NodeRankRandomnessRange:            cfg.Node.Requester.NodeRankRandomnessRange,
		OverAskForBidsFactor:               cfg.Node.Requester.OverAskForBidsFactor,
		JobSelectionPolicy: node.JobSelectionPolicy{
			Locality:            semantic.JobSelectionDataLocality(cfg.Node.Requester.JobSelectionPolicy.Locality),
			RejectStatelessJobs: cfg.Node.Requester.JobSelectionPolicy.RejectStatelessJobs,
			AcceptNetworkedJobs: cfg.Node.Requester.JobSelectionPolicy.AcceptNetworkedJobs,
			ProbeHTTP:           cfg.Node.Requester.JobSelectionPolicy.ProbeHTTP,
			ProbeExec:           cfg.Node.Requester.JobSelectionPolicy.ProbeExec,
		},
		FailureInjectionConfig:         cfg.Node.Requester.FailureInjectionConfig,
		EvalBrokerVisibilityTimeout:    time.Duration(cfg.Node.Requester.EvaluationBroker.EvalBrokerVisibilityTimeout),
		EvalBrokerInitialRetryDelay:    time.Duration(cfg.Node.Requester.EvaluationBroker.EvalBrokerInitialRetryDelay),
		EvalBrokerSubsequentRetryDelay: time.Duration(cfg.Node.Requester.EvaluationBroker.EvalBrokerSubsequentRetryDelay),
		EvalBrokerMaxRetryCount:        cfg.Node.Requester.EvaluationBroker.EvalBrokerMaxRetryCount,
		WorkerCount:                    cfg.Node.Requester.Worker.WorkerCount,
		NodeOverSubscriptionFactor:     cfg.Node.Requester.Scheduler.NodeOverSubscriptionFactor,
		WorkerEvalDequeueTimeout:       time.Duration(cfg.Node.Requester.Worker.WorkerEvalDequeueTimeout),
		WorkerEvalDequeueBaseBackoff:   time.Duration(cfg.Node.Requester.Worker.WorkerEvalDequeueBaseBackoff),
		WorkerEvalDequeueMaxBackoff:    time.Duration(cfg.Node.Requester.Worker.WorkerEvalDequeueMaxBackoff),
		SchedulerQueueBackoff:          time.Duration(cfg.Node.Requester.Scheduler.QueueBackoff),
		S3PreSignedURLExpiration:       time.Duration(cfg.Node.Requester.StorageProvider.S3.PreSignedURLExpiration),
		S3PreSignedURLDisabled:         cfg.Node.Requester.StorageProvider.S3.PreSignedURLDisabled,
		TranslationEnabled:             cfg.Node.Requester.TranslationEnabled,
		JobStore:                       jobStore,
		DefaultPublisher:               cfg.Node.Requester.DefaultPublisher,
		NodeInfoStoreTTL:               time.Duration(cfg.Node.Requester.NodeInfoStoreTTL),
	})
	if err != nil {
		return node.RequesterConfig{}, err
	}

	if cfg.Node.Requester.ManualNodeApproval {
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

func getNetworkConfig(networkStorePath string, cfg types.NetworkConfig) (node.NetworkConfig, error) {
	return node.NetworkConfig{
		Port:                     cfg.Port,
		AdvertisedAddress:        cfg.AdvertisedAddress,
		Orchestrators:            cfg.Orchestrators,
		StoreDir:                 networkStorePath,
		AuthSecret:               cfg.AuthSecret,
		ClusterName:              cfg.Cluster.Name,
		ClusterPort:              cfg.Cluster.Port,
		ClusterAdvertisedAddress: cfg.Cluster.AdvertisedAddress,
		ClusterPeers:             cfg.Cluster.Peers,
	}, nil
}

func getExecutionStore(ctx context.Context, storeCfg types.JobStoreConfig, path string) (store.ExecutionStore, error) {
	if err := storeCfg.Validate(); err != nil {
		return nil, err
	}

	switch storeCfg.Type {
	case types.BoltDB:
		return boltdb.NewStore(ctx, path)
	default:
		return nil, fmt.Errorf("unknown JobStore type: %s", storeCfg.Type)
	}
}

func getJobStore(ctx context.Context, storeCfg types.JobStoreConfig, path string) (jobstore.Store, error) {
	if err := storeCfg.Validate(); err != nil {
		return nil, err
	}

	switch storeCfg.Type {
	case types.BoltDB:
		log.Ctx(ctx).Debug().Str("Path", path).Msg("creating boltdb backed jobstore")
		return boltjobstore.NewBoltJobStore(path)
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

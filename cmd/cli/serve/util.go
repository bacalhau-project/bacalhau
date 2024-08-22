package serve

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"strconv"
	"time"

	pkgerrors "github.com/pkg/errors"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	boltjobstore "github.com/bacalhau-project/bacalhau/pkg/jobstore/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/lib/network"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/node"
)

func GetComputeConfig(
	ctx context.Context,
	cfg types2.Bacalhau,
	createExecutionStore bool,
) (node.ComputeConfig, error) {
	// TODO(review): unsure what to do here now
	/*
		totalResources, totalErr := cfg.Node.Compute.Capacity.TotalResourceLimits.ToResources()
		jobResources, jobErr := cfg.Node.Compute.Capacity.JobResourceLimits.ToResources()
		defaultResources, defaultErr := cfg.Node.Compute.Capacity.DefaultJobResourceLimits.ToResources()
		if err := errors.Join(totalErr, jobErr, defaultErr); err != nil {
			return node.ComputeConfig{}, err
		}

	*/
	var executionStore store.ExecutionStore
	if createExecutionStore {
		var err error
		executionStoreDBPath, err := cfg.ExecutionStoreFilePath()
		if err != nil {
			return node.ComputeConfig{}, err
		}
		executionStore, err = boltdb.NewStore(ctx, executionStoreDBPath)
	}

	executionsPath, err := cfg.ExecutionDir()
	if err != nil {
		return node.ComputeConfig{}, err
	}

	params := node.ComputeConfigParams{
		ExecutionStore: executionStore,

		// TODO(review): what are we setting these fields to now? We have a new set of defaults that are based on the job type.
		//TotalResourceLimits:      *totalResources,
		//JobResourceLimits:        *jobResources,
		//DefaultJobResourceLimits: *defaultResources,

		// TODO(review): assumedly we no longer support this feature?
		// IgnorePhysicalResourceLimits:          false,

		// NB(forrest): not setting these fields should result in a default populating the field by calling method
		// JobNegotiationTimeout:                 time.Duration(cfg.Node.Compute.JobTimeouts.JobNegotiationTimeout),
		// MinJobExecutionTimeout:                time.Duration(cfg.Node.Compute.JobTimeouts.MinJobExecutionTimeout),
		// MaxJobExecutionTimeout:                time.Duration(cfg.Node.Compute.JobTimeouts.MaxJobExecutionTimeout),
		// DefaultJobExecutionTimeout:            time.Duration(cfg.Node.Compute.JobTimeouts.DefaultJobExecutionTimeout),
		// JobExecutionTimeoutClientIDBypassList: cfg.Node.Compute.JobTimeouts.JobExecutionTimeoutClientIDBypassList,

		JobSelectionPolicy: node.JobSelectionPolicy{
			// Locality:            semantic.JobSelectionDataLocality(cfg.Node.Compute.JobSelection.Locality),
			// TODO(review): assumedly policy should be to accept jobs with data anywhere as we have disabled the option
			// in the config?
			Locality:            semantic.Anywhere,
			RejectStatelessJobs: cfg.JobAdmissionControl.RejectStatelessJobs,
			AcceptNetworkedJobs: cfg.JobAdmissionControl.AcceptNetworkedJobs,
			ProbeHTTP:           cfg.JobAdmissionControl.ProbeHTTP,
			ProbeExec:           cfg.JobAdmissionControl.ProbeExec,
		},

		// NB(forrest): not setting these fields should result in a default populating the field by calling method
		// LogRunningExecutionsInterval: time.Duration(cfg.Logging.LogDebugInfoInterval),
		// LogStreamBufferSize:          cfg.Node.Compute.LogStreamConfig.ChannelBufferSize,
	}

	// TODO(review): assumedly this should work this way, but not sure.
	// what is the default publisher config in the event one is not provided?
	if cfg.Publishers.Enabled(types2.KindPublisherLocal) && cfg.Publishers.HasConfig(types2.KindPublisherLocal) {
		lpcfg, err := types2.DecodeProviderConfig[types2.LocalPublisherConfig](cfg.Publishers)
		if err != nil {
			return node.ComputeConfig{}, err
		}
		params.LocalPublisher = types.LocalPublisherConfig(lpcfg)
	}

	return node.NewComputeConfigWith(executionsPath, params)
}

func GetRequesterConfig(ctx context.Context, cfg types2.Bacalhau, createJobStore bool) (node.RequesterConfig, error) {
	var err error
	var jobStore jobstore.Store
	if createJobStore {
		jobStoreDBPath, err := cfg.JobStoreFilePath()
		if err != nil {
			return node.RequesterConfig{}, err
		}
		jobStore, err = boltjobstore.NewBoltJobStore(jobStoreDBPath)
		if err != nil {
			return node.RequesterConfig{}, pkgerrors.Wrapf(err, "failed to create job store")
		}
	}
	params := node.RequesterConfigParams{
		// NB(forrest): not setting these fields should result in a default populating the field by calling method
		/*
			JobDefaults: transformer.JobDefaults{
				TotalTimeout:     time.Duration(cfg.Node.Requester.JobDefaults.TotalTimeout),
				ExecutionTimeout: time.Duration(cfg.Node.Requester.JobDefaults.ExecutionTimeout),
				QueueTimeout:     time.Duration(cfg.Node.Requester.JobDefaults.QueueTimeout),
			},

			NodeRankRandomnessRange:            cfg.Node.Requester.NodeRankRandomnessRange,
			OverAskForBidsFactor:               cfg.Node.Requester.OverAskForBidsFactor,
			FailureInjectionConfig:         cfg.Node.Requester.FailureInjectionConfig,
			EvalBrokerInitialRetryDelay:    time.Duration(cfg.Node.Requester.EvaluationBroker.EvalBrokerInitialRetryDelay),
			EvalBrokerSubsequentRetryDelay: time.Duration(cfg.Node.Requester.EvaluationBroker.EvalBrokerSubsequentRetryDelay),
			NodeOverSubscriptionFactor:     cfg.Node.Requester.Scheduler.NodeOverSubscriptionFactor,
			WorkerEvalDequeueTimeout:     time.Duration(cfg.Node.Requester.Worker.WorkerEvalDequeueTimeout),
			WorkerEvalDequeueBaseBackoff: time.Duration(cfg.Node.Requester.Worker.WorkerEvalDequeueBaseBackoff),
			WorkerEvalDequeueMaxBackoff:  time.Duration(cfg.Node.Requester.Worker.WorkerEvalDequeueMaxBackoff),
			SchedulerQueueBackoff:        time.Duration(cfg.Node.Requester.Scheduler.QueueBackoff),

			// TODO this field is never read
			NodeInfoStoreTTL:            time.Duration(cfg.Node.Requester.NodeInfoStoreTTL),
		*/
		HousekeepingBackgroundTaskInterval: time.Duration(cfg.Orchestrator.Scheduler.HousekeepingInterval),
		HousekeepingTimeoutBuffer:          time.Duration(cfg.Orchestrator.Scheduler.HousekeepingTimeout),
		JobSelectionPolicy: node.JobSelectionPolicy{
			// TODO(review): assumedly policy should be to accept jobs with data anywhere?
			Locality:            semantic.Anywhere,
			RejectStatelessJobs: cfg.JobAdmissionControl.RejectStatelessJobs,
			AcceptNetworkedJobs: cfg.JobAdmissionControl.AcceptNetworkedJobs,
			ProbeHTTP:           cfg.JobAdmissionControl.ProbeHTTP,
			ProbeExec:           cfg.JobAdmissionControl.ProbeExec,
		},
		EvalBrokerVisibilityTimeout: time.Duration(cfg.Orchestrator.EvaluationBroker.VisibilityTimeout),
		EvalBrokerMaxRetryCount:     cfg.Orchestrator.EvaluationBroker.MaxRetryCount,
		WorkerCount:                 cfg.Orchestrator.Scheduler.WorkerCount,
		TranslationEnabled:          cfg.FeatureFlags.ExecTranslation,
		JobStore:                    jobStore,
		// TODO(review): of the default job type configs it's not clear what we should be using here.
		DefaultPublisher: cfg.JobDefaults.Batch.Task.Publisher.Type,
	}

	if cfg.Publishers.Enabled(types2.KindPublisherS3) && cfg.InputSources.HasConfig(types2.KindPublisherS3) {
		s3cfg, err := types2.DecodeProviderConfig[types2.S3PublisherConfig](cfg.Publishers)
		if err != nil {
			return node.RequesterConfig{}, err
		}
		params.S3PreSignedURLExpiration = time.Duration(s3cfg.PreSignedURLExpiration)
		params.S3PreSignedURLDisabled = s3cfg.PreSignedURLDisabled
	}

	requesterConfig, err := node.NewRequesterConfigWith(params)
	if err != nil {
		return node.RequesterConfig{}, err
	}

	if cfg.Orchestrator.NodeManager.ManualApproval {
		requesterConfig.DefaultApprovalState = models.NodeMembership.PENDING
	} else {
		requesterConfig.DefaultApprovalState = models.NodeMembership.APPROVED
	}

	return requesterConfig, nil
}

func getNetworkConfig(cfg types2.Bacalhau) (node.NetworkConfig, error) {
	_, portStr, err := net.SplitHostPort(cfg.Orchestrator.Listen)
	if err != nil {
		return node.NetworkConfig{}, fmt.Errorf("failed to parse orchestrator listen address: %w", err)
	}
	listenPort, err := strconv.ParseInt(portStr, 10, 64)
	if err != nil {
		return node.NetworkConfig{}, err
	}
	storeDir, err := cfg.NetworkTransportDir()
	if err != nil {
		return node.NetworkConfig{}, err
	}
	ntwkCfg := node.NetworkConfig{
		Port:              int(listenPort),
		AdvertisedAddress: cfg.Orchestrator.Advertise,
		Orchestrators:     cfg.Compute.Orchestrators,
		StoreDir:          storeDir,
		// TODO(review): an auth secret is no longer part of the config, should we operate without one, or include it
		// in the config?
		//AuthSecret: "TODO",
		// TODO(review): a cluster name is no longer part of the config, should we operate without one, or include it
		// in the config?
		//ClusterName:              "TODO",
		ClusterAdvertisedAddress: cfg.Orchestrator.Cluster.Advertise,
		ClusterPeers:             cfg.Orchestrator.Cluster.Peers,
	}
	if cfg.Orchestrator.Cluster.Listen != "" {
		parsedURL, err := url.Parse(cfg.Orchestrator.Cluster.Listen)
		if err != nil {
			return node.NetworkConfig{}, fmt.Errorf("failed to parse cluster listen address: %w", err)
		}
		clusterListenPort, err := strconv.ParseInt(parsedURL.Port(), 10, 64)
		if err != nil {
			return node.NetworkConfig{}, err
		}
		ntwkCfg.ClusterPort = int(clusterListenPort)
	}
	return ntwkCfg, nil
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
	parsedURL, err := url.Parse(host)
	if err != nil {
		return "", err
	}
	h := parsedURL.Hostname()
	if net.ParseIP(h) == nil {
		// We should check that the value gives us an address type
		// we can use to get our IP address. If it doesn't, we should
		// panic.
		atype, ok := network.AddressTypeFromString(h)
		if !ok {
			return "", fmt.Errorf("invalid address type in Server API Host config: %s", h)
		}

		addr, err := network.GetNetworkAddress(atype, network.AllAddresses)
		if err != nil {
			return "", fmt.Errorf("failed to get network address for Server API Host: %s: %w", h, err)
		}

		if len(addr) == 0 {
			return "", fmt.Errorf("no %s addresses found for Server API Host", h)
		}

		// Use the first address
		h = addr[0]
	}

	return h, nil
}

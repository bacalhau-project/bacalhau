package serve

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func GetComputeConfig(ctx context.Context, createExecutionStore bool) (node.ComputeConfig, error) {
	panic("deprecate me")
	/*
		var cfg types.ComputeConfig
		if err := config.ForKey(types.NodeCompute, &cfg); err != nil {
			return node.ComputeConfig{}, err
		}

		totalResources, totalErr := cfg.Capacity.TotalResourceLimits.ToResources()
		queueResources, queueErr := cfg.Capacity.QueueResourceLimits.ToResources()
		jobResources, jobErr := cfg.Capacity.JobResourceLimits.ToResources()
		defaultResources, defaultErr := cfg.Capacity.DefaultJobResourceLimits.ToResources()
		if err := errors.Join(totalErr, queueErr, jobErr, defaultErr); err != nil {
			return node.ComputeConfig{}, err
		}

		var err error
		var executionStore store.ExecutionStore

		if createExecutionStore {
			executionStore, err = getExecutionStore(ctx, cfg.ExecutionStore)
			if err != nil {
				return node.ComputeConfig{}, pkgerrors.Wrapf(err, "failed to create execution store")
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
				Locality:            semantic.JobSelectionDataLocality(cfg.JobSelection.Policy.Locality),
				RejectStatelessJobs: cfg.JobSelection.Policy.RejectStatelessJobs,
				AcceptNetworkedJobs: cfg.JobSelection.Policy.AcceptNetworkedJobs,
				ProbeHTTP:           cfg.JobSelection.Policy.ProbeHTTP,
				ProbeExec:           cfg.JobSelection.Policy.ProbeExec,
			},
			LogRunningExecutionsInterval: time.Duration(cfg.Logging.LogRunningExecutionsInterval),
			LogStreamBufferSize:          cfg.LogStreamConfig.ChannelBufferSize,
			ExecutionStore:               executionStore,
			LocalPublisher:               cfg.LocalPublisher,
		})

	*/
}

func GetRequesterConfig(ctx context.Context, createJobStore bool) (node.RequesterConfig, error) {
	panic("deprecate me")
	/*
		var cfg types.RequesterConfig
		if err := config.ForKey(types.NodeRequester, &cfg); err != nil {
			return node.RequesterConfig{}, err
		}

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

	*/
}

func getNodeType(c *config.Config) (requester, compute bool, err error) {
	requester = false
	compute = false
	err = nil

	var nodeType []string
	err = c.ForKey(types.NodeType, &nodeType)
	if err != nil {
		return false, false, err
	}
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

func getIPFSConfig(c *config.Config) (types.IpfsConfig, error) {
	var ipfsConfig types.IpfsConfig
	if err := c.ForKey(types.NodeIPFS, &ipfsConfig); err != nil {
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

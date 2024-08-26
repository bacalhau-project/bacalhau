package serve

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	pkgerrors "github.com/pkg/errors"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity/system"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	boltjobstore "github.com/bacalhau-project/bacalhau/pkg/jobstore/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/node"
)

func GetComputeConfig(
	ctx context.Context,
	cfg types2.Bacalhau,
	createExecutionStore bool,
) (node.ComputeConfig, error) {
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
	systemCapacity, err := system.NewPhysicalCapacityProvider(executionsPath).GetTotalCapacity(ctx)
	if err != nil {
		return node.ComputeConfig{}, fmt.Errorf("failed to determine total system capacity: %w", err)
	}
	allocatedResources, err := scaleCapacityByAllocation(systemCapacity, cfg.Compute.AllocatedCapacity)
	if err != nil {
		return node.ComputeConfig{}, err
	}

	params := node.ComputeConfigParams{
		TotalResourceLimits:      allocatedResources,
		JobResourceLimits:        allocatedResources,
		DefaultJobResourceLimits: allocatedResources,
		ExecutionStore:           executionStore,
		JobSelectionPolicy: node.JobSelectionPolicy{
			Locality:            semantic.Anywhere,
			RejectStatelessJobs: cfg.JobAdmissionControl.RejectStatelessJobs,
			AcceptNetworkedJobs: cfg.JobAdmissionControl.AcceptNetworkedJobs,
			ProbeHTTP:           cfg.JobAdmissionControl.ProbeHTTP,
			ProbeExec:           cfg.JobAdmissionControl.ProbeExec,
		},
	}

	// if the local publisher is enabled and installed, populate params.
	// Otherwise, a default set of values will be used which are defined in NewComputeConfigWith.
	if cfg.Publishers.Enabled(models.PublisherLocal) {
		if cfg.Publishers.Local.Installed() {
			params.LocalPublisher = types.LocalPublisherConfig{
				Address:   cfg.Publishers.Local.Address,
				Port:      cfg.Publishers.Local.Port,
				Directory: cfg.Publishers.Local.Directory,
			}
		}
	}

	return node.NewComputeConfigWith(executionsPath, params)
}

func GetRequesterConfig(cfg types2.Bacalhau, createJobStore bool) (node.RequesterConfig, error) {
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
		JobDefaults:                        cfg.JobDefaults,
		HousekeepingBackgroundTaskInterval: time.Duration(cfg.Orchestrator.Scheduler.HousekeepingInterval),
		HousekeepingTimeoutBuffer:          time.Duration(cfg.Orchestrator.Scheduler.HousekeepingTimeout),
		JobSelectionPolicy: node.JobSelectionPolicy{
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
		DefaultPublisher:            cfg.DefaultPublisher.Type,
	}

	if cfg.Publishers.Enabled(models.StorageSourceS3) {
		if cfg.Publishers.S3.Installed() {
			params.S3PreSignedURLExpiration = time.Duration(cfg.Publishers.S3.PreSignedURLExpiration)
			params.S3PreSignedURLDisabled = cfg.Publishers.S3.PreSignedURLDisabled
		}
	}

	requesterConfig, err := node.NewRequesterConfigWith(params)
	if err != nil {
		return node.RequesterConfig{}, err
	}

	requesterConfig.DefaultApprovalState = models.NodeMembership.APPROVED
	if cfg.Orchestrator.NodeManager.ManualApproval {
		requesterConfig.DefaultApprovalState = models.NodeMembership.PENDING
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
		Port:                     int(listenPort),
		AdvertisedAddress:        cfg.Orchestrator.Advertise,
		Orchestrators:            cfg.Compute.Orchestrators,
		StoreDir:                 storeDir,
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

func scaleCapacityByAllocation(systemCapacity models.Resources, scaler types2.ResourceScaler) (models.Resources, error) {
	// if the system capacity is zero we should fail as it means the compute node will be unable to accept any work.
	if systemCapacity.IsZero() {
		return models.Resources{}, fmt.Errorf("system capacity is zero")
	}

	// if allocated capacity scaler is zero, return the system capacity
	if scaler.IsZero() {
		// TODO(forrest): hack because system total resources fluctuate wrt disk by several kb.
		scaler := types2.ResourceScaler{
			CPU:    "90%",
			Memory: "90%",
			Disk:   "90%",
			GPU:    "100%",
		}
		return scaler.Scale(systemCapacity)
	}

	// scale the system resources based on the allocation
	return scaler.Scale(systemCapacity)
}

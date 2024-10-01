package serve

import (
	"context"
	"fmt"
	"time"

	pkgerrors "github.com/pkg/errors"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity/system"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	legacy_types "github.com/bacalhau-project/bacalhau/pkg/config_legacy/types"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	boltjobstore "github.com/bacalhau-project/bacalhau/pkg/jobstore/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/node"
)

func GetComputeConfig(
	ctx context.Context,
	cfg types.Bacalhau,
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
		if err != nil {
			return node.ComputeConfig{}, pkgerrors.Wrapf(err, "failed to create execution store")
		}
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
		ControlPlaneSettings: legacy_types.ComputeControlPlaneConfig{
			InfoUpdateFrequency: legacy_types.Duration(cfg.Compute.Heartbeat.InfoUpdateInterval),
			ResourceUpdateFrequency: legacy_types.Duration(cfg.Compute.Heartbeat.ResourceUpdateInterval),
			HeartbeatFrequency: legacy_types.Duration(cfg.Compute.Heartbeat.Interval),
		}
	}

	// if the local publisher is enabled and installed, populate params.
	// Otherwise, a default set of values will be used which are defined in NewComputeConfigWith.
	if cfg.Publishers.IsNotDisabled(models.PublisherLocal) {
		// use the defaults, and override any values provided by the user.
		address := params.LocalPublisher.Address
		port := params.LocalPublisher.Port
		directory := params.LocalPublisher.Directory
		if cfg.Publishers.Types.Local.Address != "" {
			address = cfg.Publishers.Types.Local.Address
		}
		if cfg.Publishers.Types.Local.Port != 0 {
			port = cfg.Publishers.Types.Local.Port
		}
		if cfg.Publishers.Types.Local.Directory != "" {
			directory = cfg.Publishers.Types.Local.Directory
		}
		params.LocalPublisher = types.LocalPublisher{
			Address:   address,
			Port:      port,
			Directory: directory,
		}
	}

	return node.NewComputeConfigWith(executionsPath, params)
}

func GetRequesterConfig(cfg types.Bacalhau, createJobStore bool) (node.RequesterConfig, error) {
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
		ControlPlaneSettings: legacy_types.RequesterControlPlaneConfig{
			NodeDisconnectedAfter: legacy_types.Duration(cfg.Orchestrator.NodeManager.DisconnectTimeout),
		},
	}

	if cfg.Publishers.IsNotDisabled(models.StorageSourceS3) {
		params.S3PreSignedURLExpiration = time.Duration(cfg.Publishers.Types.S3.PreSignedURLExpiration)
		params.S3PreSignedURLDisabled = cfg.Publishers.Types.S3.PreSignedURLDisabled
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

func getNetworkConfig(cfg types.Bacalhau) (node.NetworkConfig, error) {
	storeDir, err := cfg.NetworkTransportDir()
	if err != nil {
		return node.NetworkConfig{}, err
	}
	return node.NetworkConfig{
		Host	: cfg.Orchestrator.Host,
		Port:                     cfg.Orchestrator.Port,
		AdvertisedAddress:        cfg.Orchestrator.Advertise,
		Orchestrators:            cfg.Compute.Orchestrators,
		StoreDir:                 storeDir,
		AuthSecret:               cfg.Orchestrator.AuthSecret,
		ClusterName:              cfg.Orchestrator.Cluster.Name,
		ClusterPort:              cfg.Orchestrator.Cluster.Port,
		ClusterAdvertisedAddress: cfg.Orchestrator.Cluster.Advertise,
		ClusterPeers:             cfg.Orchestrator.Cluster.Peers,
	}, nil
}
func scaleCapacityByAllocation(systemCapacity models.Resources, scaler types.ResourceScaler) (models.Resources, error) {
	// if the system capacity is zero we should fail as it means the compute node will be unable to accept any work.
	if systemCapacity.IsZero() {
		return models.Resources{}, fmt.Errorf("system capacity is zero")
	}

	// if allocated capacity scaler is zero, return the system capacity
	if scaler.IsZero() {
		return systemCapacity, nil
	}

	// scale the system resources based on the allocation
	allocatedCapacity, err := scaler.ToResource(systemCapacity)
	if err != nil {
		return models.Resources{}, fmt.Errorf("allocating system capacity: %w", err)
	}

	return *allocatedCapacity, nil
}

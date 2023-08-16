package types

import "github.com/spf13/viper"

func SetDefaults(cfg BacalhauConfig) {
viper.SetDefault(Node, cfg.Node)
viper.SetDefault(NodeAPI, cfg.Node.API)
viper.SetDefault(NodeAPIHost, cfg.Node.API.Host)
viper.SetDefault(NodeAPIPort, cfg.Node.API.Port)
viper.SetDefault(NodeLibp2p, cfg.Node.Libp2p)
viper.SetDefault(NodeLibp2pSwarmPort, cfg.Node.Libp2p.SwarmPort)
viper.SetDefault(NodeLibp2pPeerConnect, cfg.Node.Libp2p.PeerConnect)
viper.SetDefault(NodeIPFS, cfg.Node.IPFS)
viper.SetDefault(NodeIPFSConnect, cfg.Node.IPFS.Connect)
viper.SetDefault(NodeIPFSPrivateInternal, cfg.Node.IPFS.PrivateInternal)
viper.SetDefault(NodeIPFSSwarmAddresses, cfg.Node.IPFS.SwarmAddresses)
viper.SetDefault(NodeCompute, cfg.Node.Compute)
viper.SetDefault(NodeComputeCapacity, cfg.Node.Compute.Capacity)
viper.SetDefault(NodeComputeCapacityIgnorePhysicalResourceLimits, cfg.Node.Compute.Capacity.IgnorePhysicalResourceLimits)
viper.SetDefault(NodeComputeCapacityTotalResourceLimits, cfg.Node.Compute.Capacity.TotalResourceLimits)
viper.SetDefault(NodeComputeCapacityTotalResourceLimitsCPU, cfg.Node.Compute.Capacity.TotalResourceLimits.CPU)
viper.SetDefault(NodeComputeCapacityTotalResourceLimitsMemory, cfg.Node.Compute.Capacity.TotalResourceLimits.Memory)
viper.SetDefault(NodeComputeCapacityTotalResourceLimitsDisk, cfg.Node.Compute.Capacity.TotalResourceLimits.Disk)
viper.SetDefault(NodeComputeCapacityTotalResourceLimitsGPU, cfg.Node.Compute.Capacity.TotalResourceLimits.GPU)
viper.SetDefault(NodeComputeCapacityJobResourceLimits, cfg.Node.Compute.Capacity.JobResourceLimits)
viper.SetDefault(NodeComputeCapacityJobResourceLimitsCPU, cfg.Node.Compute.Capacity.JobResourceLimits.CPU)
viper.SetDefault(NodeComputeCapacityJobResourceLimitsMemory, cfg.Node.Compute.Capacity.JobResourceLimits.Memory)
viper.SetDefault(NodeComputeCapacityJobResourceLimitsDisk, cfg.Node.Compute.Capacity.JobResourceLimits.Disk)
viper.SetDefault(NodeComputeCapacityJobResourceLimitsGPU, cfg.Node.Compute.Capacity.JobResourceLimits.GPU)
viper.SetDefault(NodeComputeCapacityDefaultJobResourceLimits, cfg.Node.Compute.Capacity.DefaultJobResourceLimits)
viper.SetDefault(NodeComputeCapacityDefaultJobResourceLimitsCPU, cfg.Node.Compute.Capacity.DefaultJobResourceLimits.CPU)
viper.SetDefault(NodeComputeCapacityDefaultJobResourceLimitsMemory, cfg.Node.Compute.Capacity.DefaultJobResourceLimits.Memory)
viper.SetDefault(NodeComputeCapacityDefaultJobResourceLimitsDisk, cfg.Node.Compute.Capacity.DefaultJobResourceLimits.Disk)
viper.SetDefault(NodeComputeCapacityDefaultJobResourceLimitsGPU, cfg.Node.Compute.Capacity.DefaultJobResourceLimits.GPU)
viper.SetDefault(NodeComputeCapacityQueueResourceLimits, cfg.Node.Compute.Capacity.QueueResourceLimits)
viper.SetDefault(NodeComputeCapacityQueueResourceLimitsCPU, cfg.Node.Compute.Capacity.QueueResourceLimits.CPU)
viper.SetDefault(NodeComputeCapacityQueueResourceLimitsMemory, cfg.Node.Compute.Capacity.QueueResourceLimits.Memory)
viper.SetDefault(NodeComputeCapacityQueueResourceLimitsDisk, cfg.Node.Compute.Capacity.QueueResourceLimits.Disk)
viper.SetDefault(NodeComputeCapacityQueueResourceLimitsGPU, cfg.Node.Compute.Capacity.QueueResourceLimits.GPU)
viper.SetDefault(NodeComputeCapacityMaxJobExecutionTimeout, cfg.Node.Compute.Capacity.MaxJobExecutionTimeout.String())
viper.SetDefault(NodeComputeExecutionStore, cfg.Node.Compute.ExecutionStore)
viper.SetDefault(NodeComputeExecutionStoreType, cfg.Node.Compute.ExecutionStore.Type.String())
viper.SetDefault(NodeComputeExecutionStorePath, cfg.Node.Compute.ExecutionStore.Path)
viper.SetDefault(NodeComputeJobTimeouts, cfg.Node.Compute.JobTimeouts)
viper.SetDefault(NodeComputeJobTimeoutsJobExecutionTimeoutClientIDBypassList, cfg.Node.Compute.JobTimeouts.JobExecutionTimeoutClientIDBypassList)
viper.SetDefault(NodeComputeJobTimeoutsJobNegotiationTimeout, cfg.Node.Compute.JobTimeouts.JobNegotiationTimeout.String())
viper.SetDefault(NodeComputeJobTimeoutsMinJobExecutionTimeout, cfg.Node.Compute.JobTimeouts.MinJobExecutionTimeout.String())
viper.SetDefault(NodeComputeJobTimeoutsMaxJobExecutionTimeout, cfg.Node.Compute.JobTimeouts.MaxJobExecutionTimeout.String())
viper.SetDefault(NodeComputeJobTimeoutsDefaultJobExecutionTimeout, cfg.Node.Compute.JobTimeouts.DefaultJobExecutionTimeout.String())
viper.SetDefault(NodeComputeJobSelection, cfg.Node.Compute.JobSelection)
viper.SetDefault(NodeComputeJobSelectionLocality, cfg.Node.Compute.JobSelection.Locality.String())
viper.SetDefault(NodeComputeJobSelectionRejectStatelessJobs, cfg.Node.Compute.JobSelection.RejectStatelessJobs)
viper.SetDefault(NodeComputeJobSelectionAcceptNetworkedJobs, cfg.Node.Compute.JobSelection.AcceptNetworkedJobs)
viper.SetDefault(NodeComputeJobSelectionProbeHTTP, cfg.Node.Compute.JobSelection.ProbeHTTP)
viper.SetDefault(NodeComputeJobSelectionProbeExec, cfg.Node.Compute.JobSelection.ProbeExec)
viper.SetDefault(NodeComputeQueue, cfg.Node.Compute.Queue)
viper.SetDefault(NodeComputeQueueExecutorBufferBackoffDuration, cfg.Node.Compute.Queue.ExecutorBufferBackoffDuration.String())
viper.SetDefault(NodeComputeLogging, cfg.Node.Compute.Logging)
viper.SetDefault(NodeComputeLoggingLogRunningExecutionsInterval, cfg.Node.Compute.Logging.LogRunningExecutionsInterval.String())
viper.SetDefault(NodeRequester, cfg.Node.Requester)
viper.SetDefault(NodeRequesterExternalVerifierHook, cfg.Node.Requester.ExternalVerifierHook)
viper.SetDefault(NodeRequesterJobSelectionPolicy, cfg.Node.Requester.JobSelectionPolicy)
viper.SetDefault(NodeRequesterJobSelectionPolicyLocality, cfg.Node.Requester.JobSelectionPolicy.Locality.String())
viper.SetDefault(NodeRequesterJobSelectionPolicyRejectStatelessJobs, cfg.Node.Requester.JobSelectionPolicy.RejectStatelessJobs)
viper.SetDefault(NodeRequesterJobSelectionPolicyAcceptNetworkedJobs, cfg.Node.Requester.JobSelectionPolicy.AcceptNetworkedJobs)
viper.SetDefault(NodeRequesterJobSelectionPolicyProbeHTTP, cfg.Node.Requester.JobSelectionPolicy.ProbeHTTP)
viper.SetDefault(NodeRequesterJobSelectionPolicyProbeExec, cfg.Node.Requester.JobSelectionPolicy.ProbeExec)
viper.SetDefault(NodeRequesterJobStore, cfg.Node.Requester.JobStore)
viper.SetDefault(NodeRequesterJobStoreType, cfg.Node.Requester.JobStore.Type.String())
viper.SetDefault(NodeRequesterJobStorePath, cfg.Node.Requester.JobStore.Path)
viper.SetDefault(NodeRequesterHousekeepingBackgroundTaskInterval, cfg.Node.Requester.HousekeepingBackgroundTaskInterval.String())
viper.SetDefault(NodeRequesterNodeRankRandomnessRange, cfg.Node.Requester.NodeRankRandomnessRange)
viper.SetDefault(NodeRequesterOverAskForBidsFactor, cfg.Node.Requester.OverAskForBidsFactor)
viper.SetDefault(NodeRequesterFailureInjectionConfig, cfg.Node.Requester.FailureInjectionConfig)
viper.SetDefault(NodeRequesterFailureInjectionConfigIsBadActor, cfg.Node.Requester.FailureInjectionConfig.IsBadActor)
viper.SetDefault(NodeRequesterEvaluationBroker, cfg.Node.Requester.EvaluationBroker)
viper.SetDefault(NodeRequesterEvaluationBrokerEvalBrokerVisibilityTimeout, cfg.Node.Requester.EvaluationBroker.EvalBrokerVisibilityTimeout.String())
viper.SetDefault(NodeRequesterEvaluationBrokerEvalBrokerInitialRetryDelay, cfg.Node.Requester.EvaluationBroker.EvalBrokerInitialRetryDelay.String())
viper.SetDefault(NodeRequesterEvaluationBrokerEvalBrokerSubsequentRetryDelay, cfg.Node.Requester.EvaluationBroker.EvalBrokerSubsequentRetryDelay.String())
viper.SetDefault(NodeRequesterEvaluationBrokerEvalBrokerMaxRetryCount, cfg.Node.Requester.EvaluationBroker.EvalBrokerMaxRetryCount)
viper.SetDefault(NodeRequesterWorker, cfg.Node.Requester.Worker)
viper.SetDefault(NodeRequesterWorkerWorkerCount, cfg.Node.Requester.Worker.WorkerCount)
viper.SetDefault(NodeRequesterWorkerWorkerEvalDequeueTimeout, cfg.Node.Requester.Worker.WorkerEvalDequeueTimeout.String())
viper.SetDefault(NodeRequesterWorkerWorkerEvalDequeueBaseBackoff, cfg.Node.Requester.Worker.WorkerEvalDequeueBaseBackoff.String())
viper.SetDefault(NodeRequesterWorkerWorkerEvalDequeueMaxBackoff, cfg.Node.Requester.Worker.WorkerEvalDequeueMaxBackoff.String())
viper.SetDefault(NodeRequesterTimeouts, cfg.Node.Requester.Timeouts)
viper.SetDefault(NodeRequesterTimeoutsMinJobExecutionTimeout, cfg.Node.Requester.Timeouts.MinJobExecutionTimeout.String())
viper.SetDefault(NodeRequesterTimeoutsDefaultJobExecutionTimeout, cfg.Node.Requester.Timeouts.DefaultJobExecutionTimeout.String())
viper.SetDefault(NodeBootstrapAddresses, cfg.Node.BootstrapAddresses)
viper.SetDefault(NodeDownloadURLRequestRetries, cfg.Node.DownloadURLRequestRetries)
viper.SetDefault(NodeDownloadURLRequestTimeout, cfg.Node.DownloadURLRequestTimeout.String())
viper.SetDefault(NodeVolumeSizeRequestTimeout, cfg.Node.VolumeSizeRequestTimeout.String())
viper.SetDefault(NodeExecutorPluginPath, cfg.Node.ExecutorPluginPath)
viper.SetDefault(NodeComputeStoragePath, cfg.Node.ComputeStoragePath)
viper.SetDefault(NodeLoggingMode, cfg.Node.LoggingMode)
viper.SetDefault(NodeType, cfg.Node.Type)
viper.SetDefault(NodeEstuaryAPIKey, cfg.Node.EstuaryAPIKey)
viper.SetDefault(NodeAllowListedLocalPaths, cfg.Node.AllowListedLocalPaths)
viper.SetDefault(NodeDisabledFeatures, cfg.Node.DisabledFeatures)
viper.SetDefault(NodeDisabledFeaturesEngines, cfg.Node.DisabledFeatures.Engines)
viper.SetDefault(NodeDisabledFeaturesPublishers, cfg.Node.DisabledFeatures.Publishers)
viper.SetDefault(NodeDisabledFeaturesStorages, cfg.Node.DisabledFeatures.Storages)
viper.SetDefault(NodeLabels, cfg.Node.Labels)
viper.SetDefault(NodeUser, cfg.User)
viper.SetDefault(NodeUserUserKeyPath, cfg.User.UserKeyPath)
viper.SetDefault(NodeUserLibp2pKeyPath, cfg.User.Libp2pKeyPath)
viper.SetDefault(NodeMetrics, cfg.Metrics)
viper.SetDefault(NodeMetricsLibp2pTracerPath, cfg.Metrics.Libp2pTracerPath)
viper.SetDefault(NodeMetricsEventTracerPath, cfg.Metrics.EventTracerPath)
}

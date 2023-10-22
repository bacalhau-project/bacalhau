// CODE GENERATED BY pkg/config/types/gen_viper DO NOT EDIT

package types

import "github.com/spf13/viper"

func SetDefaults(cfg BacalhauConfig) {
	viper.SetDefault(Node, cfg.Node)
	viper.SetDefault(NodeClientAPI, cfg.Node.ClientAPI)
	viper.SetDefault(NodeClientAPIHost, cfg.Node.ClientAPI.Host)
	viper.SetDefault(NodeClientAPIPort, cfg.Node.ClientAPI.Port)
	viper.SetDefault(NodeClientAPITLS, cfg.Node.ClientAPI.TLS)
	viper.SetDefault(NodeClientAPITLSAutoCert, cfg.Node.ClientAPI.TLS.AutoCert)
	viper.SetDefault(NodeClientAPITLSAutoCertCachePath, cfg.Node.ClientAPI.TLS.AutoCertCachePath)
	viper.SetDefault(NodeClientAPITLSServerCertificate, cfg.Node.ClientAPI.TLS.ServerCertificate)
	viper.SetDefault(NodeClientAPITLSServerKey, cfg.Node.ClientAPI.TLS.ServerKey)
	viper.SetDefault(NodeServerAPI, cfg.Node.ServerAPI)
	viper.SetDefault(NodeServerAPIHost, cfg.Node.ServerAPI.Host)
	viper.SetDefault(NodeServerAPIPort, cfg.Node.ServerAPI.Port)
	viper.SetDefault(NodeServerAPITLS, cfg.Node.ServerAPI.TLS)
	viper.SetDefault(NodeServerAPITLSAutoCert, cfg.Node.ServerAPI.TLS.AutoCert)
	viper.SetDefault(NodeServerAPITLSAutoCertCachePath, cfg.Node.ServerAPI.TLS.AutoCertCachePath)
	viper.SetDefault(NodeServerAPITLSServerCertificate, cfg.Node.ServerAPI.TLS.ServerCertificate)
	viper.SetDefault(NodeServerAPITLSServerKey, cfg.Node.ServerAPI.TLS.ServerKey)
	viper.SetDefault(NodeLibp2p, cfg.Node.Libp2p)
	viper.SetDefault(NodeLibp2pSwarmPort, cfg.Node.Libp2p.SwarmPort)
	viper.SetDefault(NodeLibp2pPeerConnect, cfg.Node.Libp2p.PeerConnect)
	viper.SetDefault(NodeIPFS, cfg.Node.IPFS)
	viper.SetDefault(NodeIPFSConnect, cfg.Node.IPFS.Connect)
	viper.SetDefault(NodeIPFSPrivateInternal, cfg.Node.IPFS.PrivateInternal)
	viper.SetDefault(NodeIPFSSwarmAddresses, cfg.Node.IPFS.SwarmAddresses)
	viper.SetDefault(NodeIPFSSwarmKeyPath, cfg.Node.IPFS.SwarmKeyPath)
	viper.SetDefault(NodeIPFSServePath, cfg.Node.IPFS.ServePath)
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
	viper.SetDefault(NodeRequesterJobDefaults, cfg.Node.Requester.JobDefaults)
	viper.SetDefault(NodeRequesterJobDefaultsExecutionTimeout, cfg.Node.Requester.JobDefaults.ExecutionTimeout.String())
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
	viper.SetDefault(User, cfg.User)
	viper.SetDefault(UserKeyPath, cfg.User.KeyPath)
	viper.SetDefault(UserLibp2pKeyPath, cfg.User.Libp2pKeyPath)
	viper.SetDefault(UserUserID, cfg.User.UserID)
	viper.SetDefault(Metrics, cfg.Metrics)
	viper.SetDefault(MetricsLibp2pTracerPath, cfg.Metrics.Libp2pTracerPath)
	viper.SetDefault(MetricsEventTracerPath, cfg.Metrics.EventTracerPath)
	viper.SetDefault(Update, cfg.Update)
	viper.SetDefault(UpdateSkipChecks, cfg.Update.SkipChecks)
	viper.SetDefault(UpdateCheckStatePath, cfg.Update.CheckStatePath)
	viper.SetDefault(UpdateCheckFrequency, cfg.Update.CheckFrequency.String())
}

func Set(cfg BacalhauConfig) {
	viper.Set(Node, cfg.Node)
	viper.Set(NodeClientAPI, cfg.Node.ClientAPI)
	viper.Set(NodeClientAPIHost, cfg.Node.ClientAPI.Host)
	viper.Set(NodeClientAPIPort, cfg.Node.ClientAPI.Port)
	viper.Set(NodeClientAPITLS, cfg.Node.ClientAPI.TLS)
	viper.Set(NodeClientAPITLSAutoCert, cfg.Node.ClientAPI.TLS.AutoCert)
	viper.Set(NodeClientAPITLSAutoCertCachePath, cfg.Node.ClientAPI.TLS.AutoCertCachePath)
	viper.Set(NodeClientAPITLSServerCertificate, cfg.Node.ClientAPI.TLS.ServerCertificate)
	viper.Set(NodeClientAPITLSServerKey, cfg.Node.ClientAPI.TLS.ServerKey)
	viper.Set(NodeServerAPI, cfg.Node.ServerAPI)
	viper.Set(NodeServerAPIHost, cfg.Node.ServerAPI.Host)
	viper.Set(NodeServerAPIPort, cfg.Node.ServerAPI.Port)
	viper.Set(NodeServerAPITLS, cfg.Node.ServerAPI.TLS)
	viper.Set(NodeServerAPITLSAutoCert, cfg.Node.ServerAPI.TLS.AutoCert)
	viper.Set(NodeServerAPITLSAutoCertCachePath, cfg.Node.ServerAPI.TLS.AutoCertCachePath)
	viper.Set(NodeServerAPITLSServerCertificate, cfg.Node.ServerAPI.TLS.ServerCertificate)
	viper.Set(NodeServerAPITLSServerKey, cfg.Node.ServerAPI.TLS.ServerKey)
	viper.Set(NodeLibp2p, cfg.Node.Libp2p)
	viper.Set(NodeLibp2pSwarmPort, cfg.Node.Libp2p.SwarmPort)
	viper.Set(NodeLibp2pPeerConnect, cfg.Node.Libp2p.PeerConnect)
	viper.Set(NodeIPFS, cfg.Node.IPFS)
	viper.Set(NodeIPFSConnect, cfg.Node.IPFS.Connect)
	viper.Set(NodeIPFSPrivateInternal, cfg.Node.IPFS.PrivateInternal)
	viper.Set(NodeIPFSSwarmAddresses, cfg.Node.IPFS.SwarmAddresses)
	viper.Set(NodeIPFSSwarmKeyPath, cfg.Node.IPFS.SwarmKeyPath)
	viper.Set(NodeIPFSServePath, cfg.Node.IPFS.ServePath)
	viper.Set(NodeCompute, cfg.Node.Compute)
	viper.Set(NodeComputeCapacity, cfg.Node.Compute.Capacity)
	viper.Set(NodeComputeCapacityIgnorePhysicalResourceLimits, cfg.Node.Compute.Capacity.IgnorePhysicalResourceLimits)
	viper.Set(NodeComputeCapacityTotalResourceLimits, cfg.Node.Compute.Capacity.TotalResourceLimits)
	viper.Set(NodeComputeCapacityTotalResourceLimitsCPU, cfg.Node.Compute.Capacity.TotalResourceLimits.CPU)
	viper.Set(NodeComputeCapacityTotalResourceLimitsMemory, cfg.Node.Compute.Capacity.TotalResourceLimits.Memory)
	viper.Set(NodeComputeCapacityTotalResourceLimitsDisk, cfg.Node.Compute.Capacity.TotalResourceLimits.Disk)
	viper.Set(NodeComputeCapacityTotalResourceLimitsGPU, cfg.Node.Compute.Capacity.TotalResourceLimits.GPU)
	viper.Set(NodeComputeCapacityJobResourceLimits, cfg.Node.Compute.Capacity.JobResourceLimits)
	viper.Set(NodeComputeCapacityJobResourceLimitsCPU, cfg.Node.Compute.Capacity.JobResourceLimits.CPU)
	viper.Set(NodeComputeCapacityJobResourceLimitsMemory, cfg.Node.Compute.Capacity.JobResourceLimits.Memory)
	viper.Set(NodeComputeCapacityJobResourceLimitsDisk, cfg.Node.Compute.Capacity.JobResourceLimits.Disk)
	viper.Set(NodeComputeCapacityJobResourceLimitsGPU, cfg.Node.Compute.Capacity.JobResourceLimits.GPU)
	viper.Set(NodeComputeCapacityDefaultJobResourceLimits, cfg.Node.Compute.Capacity.DefaultJobResourceLimits)
	viper.Set(NodeComputeCapacityDefaultJobResourceLimitsCPU, cfg.Node.Compute.Capacity.DefaultJobResourceLimits.CPU)
	viper.Set(NodeComputeCapacityDefaultJobResourceLimitsMemory, cfg.Node.Compute.Capacity.DefaultJobResourceLimits.Memory)
	viper.Set(NodeComputeCapacityDefaultJobResourceLimitsDisk, cfg.Node.Compute.Capacity.DefaultJobResourceLimits.Disk)
	viper.Set(NodeComputeCapacityDefaultJobResourceLimitsGPU, cfg.Node.Compute.Capacity.DefaultJobResourceLimits.GPU)
	viper.Set(NodeComputeCapacityQueueResourceLimits, cfg.Node.Compute.Capacity.QueueResourceLimits)
	viper.Set(NodeComputeCapacityQueueResourceLimitsCPU, cfg.Node.Compute.Capacity.QueueResourceLimits.CPU)
	viper.Set(NodeComputeCapacityQueueResourceLimitsMemory, cfg.Node.Compute.Capacity.QueueResourceLimits.Memory)
	viper.Set(NodeComputeCapacityQueueResourceLimitsDisk, cfg.Node.Compute.Capacity.QueueResourceLimits.Disk)
	viper.Set(NodeComputeCapacityQueueResourceLimitsGPU, cfg.Node.Compute.Capacity.QueueResourceLimits.GPU)
	viper.Set(NodeComputeExecutionStore, cfg.Node.Compute.ExecutionStore)
	viper.Set(NodeComputeExecutionStoreType, cfg.Node.Compute.ExecutionStore.Type.String())
	viper.Set(NodeComputeExecutionStorePath, cfg.Node.Compute.ExecutionStore.Path)
	viper.Set(NodeComputeJobTimeouts, cfg.Node.Compute.JobTimeouts)
	viper.Set(NodeComputeJobTimeoutsJobExecutionTimeoutClientIDBypassList, cfg.Node.Compute.JobTimeouts.JobExecutionTimeoutClientIDBypassList)
	viper.Set(NodeComputeJobTimeoutsJobNegotiationTimeout, cfg.Node.Compute.JobTimeouts.JobNegotiationTimeout.String())
	viper.Set(NodeComputeJobTimeoutsMinJobExecutionTimeout, cfg.Node.Compute.JobTimeouts.MinJobExecutionTimeout.String())
	viper.Set(NodeComputeJobTimeoutsMaxJobExecutionTimeout, cfg.Node.Compute.JobTimeouts.MaxJobExecutionTimeout.String())
	viper.Set(NodeComputeJobTimeoutsDefaultJobExecutionTimeout, cfg.Node.Compute.JobTimeouts.DefaultJobExecutionTimeout.String())
	viper.Set(NodeComputeJobSelection, cfg.Node.Compute.JobSelection)
	viper.Set(NodeComputeJobSelectionLocality, cfg.Node.Compute.JobSelection.Locality.String())
	viper.Set(NodeComputeJobSelectionRejectStatelessJobs, cfg.Node.Compute.JobSelection.RejectStatelessJobs)
	viper.Set(NodeComputeJobSelectionAcceptNetworkedJobs, cfg.Node.Compute.JobSelection.AcceptNetworkedJobs)
	viper.Set(NodeComputeJobSelectionProbeHTTP, cfg.Node.Compute.JobSelection.ProbeHTTP)
	viper.Set(NodeComputeJobSelectionProbeExec, cfg.Node.Compute.JobSelection.ProbeExec)
	viper.Set(NodeComputeQueue, cfg.Node.Compute.Queue)
	viper.Set(NodeComputeQueueExecutorBufferBackoffDuration, cfg.Node.Compute.Queue.ExecutorBufferBackoffDuration.String())
	viper.Set(NodeComputeLogging, cfg.Node.Compute.Logging)
	viper.Set(NodeComputeLoggingLogRunningExecutionsInterval, cfg.Node.Compute.Logging.LogRunningExecutionsInterval.String())
	viper.Set(NodeRequester, cfg.Node.Requester)
	viper.Set(NodeRequesterJobDefaults, cfg.Node.Requester.JobDefaults)
	viper.Set(NodeRequesterJobDefaultsExecutionTimeout, cfg.Node.Requester.JobDefaults.ExecutionTimeout.String())
	viper.Set(NodeRequesterExternalVerifierHook, cfg.Node.Requester.ExternalVerifierHook)
	viper.Set(NodeRequesterJobSelectionPolicy, cfg.Node.Requester.JobSelectionPolicy)
	viper.Set(NodeRequesterJobSelectionPolicyLocality, cfg.Node.Requester.JobSelectionPolicy.Locality.String())
	viper.Set(NodeRequesterJobSelectionPolicyRejectStatelessJobs, cfg.Node.Requester.JobSelectionPolicy.RejectStatelessJobs)
	viper.Set(NodeRequesterJobSelectionPolicyAcceptNetworkedJobs, cfg.Node.Requester.JobSelectionPolicy.AcceptNetworkedJobs)
	viper.Set(NodeRequesterJobSelectionPolicyProbeHTTP, cfg.Node.Requester.JobSelectionPolicy.ProbeHTTP)
	viper.Set(NodeRequesterJobSelectionPolicyProbeExec, cfg.Node.Requester.JobSelectionPolicy.ProbeExec)
	viper.Set(NodeRequesterJobStore, cfg.Node.Requester.JobStore)
	viper.Set(NodeRequesterJobStoreType, cfg.Node.Requester.JobStore.Type.String())
	viper.Set(NodeRequesterJobStorePath, cfg.Node.Requester.JobStore.Path)
	viper.Set(NodeRequesterHousekeepingBackgroundTaskInterval, cfg.Node.Requester.HousekeepingBackgroundTaskInterval.String())
	viper.Set(NodeRequesterNodeRankRandomnessRange, cfg.Node.Requester.NodeRankRandomnessRange)
	viper.Set(NodeRequesterOverAskForBidsFactor, cfg.Node.Requester.OverAskForBidsFactor)
	viper.Set(NodeRequesterFailureInjectionConfig, cfg.Node.Requester.FailureInjectionConfig)
	viper.Set(NodeRequesterFailureInjectionConfigIsBadActor, cfg.Node.Requester.FailureInjectionConfig.IsBadActor)
	viper.Set(NodeRequesterEvaluationBroker, cfg.Node.Requester.EvaluationBroker)
	viper.Set(NodeRequesterEvaluationBrokerEvalBrokerVisibilityTimeout, cfg.Node.Requester.EvaluationBroker.EvalBrokerVisibilityTimeout.String())
	viper.Set(NodeRequesterEvaluationBrokerEvalBrokerInitialRetryDelay, cfg.Node.Requester.EvaluationBroker.EvalBrokerInitialRetryDelay.String())
	viper.Set(NodeRequesterEvaluationBrokerEvalBrokerSubsequentRetryDelay, cfg.Node.Requester.EvaluationBroker.EvalBrokerSubsequentRetryDelay.String())
	viper.Set(NodeRequesterEvaluationBrokerEvalBrokerMaxRetryCount, cfg.Node.Requester.EvaluationBroker.EvalBrokerMaxRetryCount)
	viper.Set(NodeRequesterWorker, cfg.Node.Requester.Worker)
	viper.Set(NodeRequesterWorkerWorkerCount, cfg.Node.Requester.Worker.WorkerCount)
	viper.Set(NodeRequesterWorkerWorkerEvalDequeueTimeout, cfg.Node.Requester.Worker.WorkerEvalDequeueTimeout.String())
	viper.Set(NodeRequesterWorkerWorkerEvalDequeueBaseBackoff, cfg.Node.Requester.Worker.WorkerEvalDequeueBaseBackoff.String())
	viper.Set(NodeRequesterWorkerWorkerEvalDequeueMaxBackoff, cfg.Node.Requester.Worker.WorkerEvalDequeueMaxBackoff.String())
	viper.Set(NodeBootstrapAddresses, cfg.Node.BootstrapAddresses)
	viper.Set(NodeDownloadURLRequestRetries, cfg.Node.DownloadURLRequestRetries)
	viper.Set(NodeDownloadURLRequestTimeout, cfg.Node.DownloadURLRequestTimeout.String())
	viper.Set(NodeVolumeSizeRequestTimeout, cfg.Node.VolumeSizeRequestTimeout.String())
	viper.Set(NodeExecutorPluginPath, cfg.Node.ExecutorPluginPath)
	viper.Set(NodeComputeStoragePath, cfg.Node.ComputeStoragePath)
	viper.Set(NodeLoggingMode, cfg.Node.LoggingMode)
	viper.Set(NodeType, cfg.Node.Type)
	viper.Set(NodeEstuaryAPIKey, cfg.Node.EstuaryAPIKey)
	viper.Set(NodeAllowListedLocalPaths, cfg.Node.AllowListedLocalPaths)
	viper.Set(NodeDisabledFeatures, cfg.Node.DisabledFeatures)
	viper.Set(NodeDisabledFeaturesEngines, cfg.Node.DisabledFeatures.Engines)
	viper.Set(NodeDisabledFeaturesPublishers, cfg.Node.DisabledFeatures.Publishers)
	viper.Set(NodeDisabledFeaturesStorages, cfg.Node.DisabledFeatures.Storages)
	viper.Set(NodeLabels, cfg.Node.Labels)
	viper.Set(User, cfg.User)
	viper.Set(UserKeyPath, cfg.User.KeyPath)
	viper.Set(UserLibp2pKeyPath, cfg.User.Libp2pKeyPath)
	viper.Set(UserUserID, cfg.User.UserID)
	viper.Set(Metrics, cfg.Metrics)
	viper.Set(MetricsLibp2pTracerPath, cfg.Metrics.Libp2pTracerPath)
	viper.Set(MetricsEventTracerPath, cfg.Metrics.EventTracerPath)
	viper.Set(Update, cfg.Update)
	viper.Set(UpdateSkipChecks, cfg.Update.SkipChecks)
	viper.Set(UpdateCheckStatePath, cfg.Update.CheckStatePath)
	viper.Set(UpdateCheckFrequency, cfg.Update.CheckFrequency.String())
}

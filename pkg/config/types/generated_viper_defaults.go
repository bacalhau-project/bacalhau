// CODE GENERATED BY pkg/config/types/gen_viper DO NOT EDIT

package types

import "github.com/spf13/viper"

type SetOption func(p *SetParams)

func WithViper(v *viper.Viper) SetOption {
	return func(p *SetParams) {
		p.Viper = v
	}
}

type SetParams struct {
	Viper *viper.Viper
}

func SetDefaults(cfg BacalhauConfig, opts ...SetOption) {
	p := &SetParams{
		Viper: viper.GetViper(),
	}
	for _, opt := range opts {
		opt(p)
	}

	p.Viper.SetDefault(Node, cfg.Node)
	p.Viper.SetDefault(NodeName, cfg.Node.Name)
	p.Viper.SetDefault(NodeNameProvider, cfg.Node.NameProvider)
	p.Viper.SetDefault(NodeClientAPI, cfg.Node.ClientAPI)
	p.Viper.SetDefault(NodeClientAPIHost, cfg.Node.ClientAPI.Host)
	p.Viper.SetDefault(NodeClientAPIPort, cfg.Node.ClientAPI.Port)
	p.Viper.SetDefault(NodeClientAPIClientTLS, cfg.Node.ClientAPI.ClientTLS)
	p.Viper.SetDefault(NodeClientAPIClientTLSUseTLS, cfg.Node.ClientAPI.ClientTLS.UseTLS)
	p.Viper.SetDefault(NodeClientAPIClientTLSCACert, cfg.Node.ClientAPI.ClientTLS.CACert)
	p.Viper.SetDefault(NodeClientAPIClientTLSInsecure, cfg.Node.ClientAPI.ClientTLS.Insecure)
	p.Viper.SetDefault(NodeClientAPITLS, cfg.Node.ClientAPI.TLS)
	p.Viper.SetDefault(NodeClientAPITLSAutoCert, cfg.Node.ClientAPI.TLS.AutoCert)
	p.Viper.SetDefault(NodeClientAPITLSAutoCertCachePath, cfg.Node.ClientAPI.TLS.AutoCertCachePath)
	p.Viper.SetDefault(NodeClientAPITLSServerCertificate, cfg.Node.ClientAPI.TLS.ServerCertificate)
	p.Viper.SetDefault(NodeClientAPITLSServerKey, cfg.Node.ClientAPI.TLS.ServerKey)
	p.Viper.SetDefault(NodeClientAPITLSSelfSigned, cfg.Node.ClientAPI.TLS.SelfSigned)
	p.Viper.SetDefault(NodeServerAPI, cfg.Node.ServerAPI)
	p.Viper.SetDefault(NodeServerAPIHost, cfg.Node.ServerAPI.Host)
	p.Viper.SetDefault(NodeServerAPIPort, cfg.Node.ServerAPI.Port)
	p.Viper.SetDefault(NodeServerAPIClientTLS, cfg.Node.ServerAPI.ClientTLS)
	p.Viper.SetDefault(NodeServerAPIClientTLSUseTLS, cfg.Node.ServerAPI.ClientTLS.UseTLS)
	p.Viper.SetDefault(NodeServerAPIClientTLSCACert, cfg.Node.ServerAPI.ClientTLS.CACert)
	p.Viper.SetDefault(NodeServerAPIClientTLSInsecure, cfg.Node.ServerAPI.ClientTLS.Insecure)
	p.Viper.SetDefault(NodeServerAPITLS, cfg.Node.ServerAPI.TLS)
	p.Viper.SetDefault(NodeServerAPITLSAutoCert, cfg.Node.ServerAPI.TLS.AutoCert)
	p.Viper.SetDefault(NodeServerAPITLSAutoCertCachePath, cfg.Node.ServerAPI.TLS.AutoCertCachePath)
	p.Viper.SetDefault(NodeServerAPITLSServerCertificate, cfg.Node.ServerAPI.TLS.ServerCertificate)
	p.Viper.SetDefault(NodeServerAPITLSServerKey, cfg.Node.ServerAPI.TLS.ServerKey)
	p.Viper.SetDefault(NodeServerAPITLSSelfSigned, cfg.Node.ServerAPI.TLS.SelfSigned)
	p.Viper.SetDefault(NodeIPFS, cfg.Node.IPFS)
	p.Viper.SetDefault(NodeIPFSConnect, cfg.Node.IPFS.Connect)
	p.Viper.SetDefault(NodeCompute, cfg.Node.Compute)
	p.Viper.SetDefault(NodeComputeCapacity, cfg.Node.Compute.Capacity)
	p.Viper.SetDefault(NodeComputeCapacityIgnorePhysicalResourceLimits, cfg.Node.Compute.Capacity.IgnorePhysicalResourceLimits)
	p.Viper.SetDefault(NodeComputeCapacityTotalResourceLimits, cfg.Node.Compute.Capacity.TotalResourceLimits)
	p.Viper.SetDefault(NodeComputeCapacityTotalResourceLimitsCPU, cfg.Node.Compute.Capacity.TotalResourceLimits.CPU)
	p.Viper.SetDefault(NodeComputeCapacityTotalResourceLimitsMemory, cfg.Node.Compute.Capacity.TotalResourceLimits.Memory)
	p.Viper.SetDefault(NodeComputeCapacityTotalResourceLimitsDisk, cfg.Node.Compute.Capacity.TotalResourceLimits.Disk)
	p.Viper.SetDefault(NodeComputeCapacityTotalResourceLimitsGPU, cfg.Node.Compute.Capacity.TotalResourceLimits.GPU)
	p.Viper.SetDefault(NodeComputeCapacityJobResourceLimits, cfg.Node.Compute.Capacity.JobResourceLimits)
	p.Viper.SetDefault(NodeComputeCapacityJobResourceLimitsCPU, cfg.Node.Compute.Capacity.JobResourceLimits.CPU)
	p.Viper.SetDefault(NodeComputeCapacityJobResourceLimitsMemory, cfg.Node.Compute.Capacity.JobResourceLimits.Memory)
	p.Viper.SetDefault(NodeComputeCapacityJobResourceLimitsDisk, cfg.Node.Compute.Capacity.JobResourceLimits.Disk)
	p.Viper.SetDefault(NodeComputeCapacityJobResourceLimitsGPU, cfg.Node.Compute.Capacity.JobResourceLimits.GPU)
	p.Viper.SetDefault(NodeComputeCapacityDefaultJobResourceLimits, cfg.Node.Compute.Capacity.DefaultJobResourceLimits)
	p.Viper.SetDefault(NodeComputeCapacityDefaultJobResourceLimitsCPU, cfg.Node.Compute.Capacity.DefaultJobResourceLimits.CPU)
	p.Viper.SetDefault(NodeComputeCapacityDefaultJobResourceLimitsMemory, cfg.Node.Compute.Capacity.DefaultJobResourceLimits.Memory)
	p.Viper.SetDefault(NodeComputeCapacityDefaultJobResourceLimitsDisk, cfg.Node.Compute.Capacity.DefaultJobResourceLimits.Disk)
	p.Viper.SetDefault(NodeComputeCapacityDefaultJobResourceLimitsGPU, cfg.Node.Compute.Capacity.DefaultJobResourceLimits.GPU)
	p.Viper.SetDefault(NodeComputeExecutionStore, cfg.Node.Compute.ExecutionStore)
	p.Viper.SetDefault(NodeComputeExecutionStoreType, cfg.Node.Compute.ExecutionStore.Type)
	p.Viper.SetDefault(NodeComputeExecutionStorePath, cfg.Node.Compute.ExecutionStore.Path)
	p.Viper.SetDefault(NodeComputeJobTimeouts, cfg.Node.Compute.JobTimeouts)
	p.Viper.SetDefault(NodeComputeJobTimeoutsJobExecutionTimeoutClientIDBypassList, cfg.Node.Compute.JobTimeouts.JobExecutionTimeoutClientIDBypassList)
	p.Viper.SetDefault(NodeComputeJobTimeoutsJobNegotiationTimeout, cfg.Node.Compute.JobTimeouts.JobNegotiationTimeout.AsTimeDuration())
	p.Viper.SetDefault(NodeComputeJobTimeoutsMinJobExecutionTimeout, cfg.Node.Compute.JobTimeouts.MinJobExecutionTimeout.AsTimeDuration())
	p.Viper.SetDefault(NodeComputeJobTimeoutsMaxJobExecutionTimeout, cfg.Node.Compute.JobTimeouts.MaxJobExecutionTimeout.AsTimeDuration())
	p.Viper.SetDefault(NodeComputeJobTimeoutsDefaultJobExecutionTimeout, cfg.Node.Compute.JobTimeouts.DefaultJobExecutionTimeout.AsTimeDuration())
	p.Viper.SetDefault(NodeComputeJobSelection, cfg.Node.Compute.JobSelection)
	p.Viper.SetDefault(NodeComputeJobSelectionLocality, cfg.Node.Compute.JobSelection.Locality)
	p.Viper.SetDefault(NodeComputeJobSelectionRejectStatelessJobs, cfg.Node.Compute.JobSelection.RejectStatelessJobs)
	p.Viper.SetDefault(NodeComputeJobSelectionAcceptNetworkedJobs, cfg.Node.Compute.JobSelection.AcceptNetworkedJobs)
	p.Viper.SetDefault(NodeComputeJobSelectionProbeHTTP, cfg.Node.Compute.JobSelection.ProbeHTTP)
	p.Viper.SetDefault(NodeComputeJobSelectionProbeExec, cfg.Node.Compute.JobSelection.ProbeExec)
	p.Viper.SetDefault(NodeComputeLogging, cfg.Node.Compute.Logging)
	p.Viper.SetDefault(NodeComputeLoggingLogRunningExecutionsInterval, cfg.Node.Compute.Logging.LogRunningExecutionsInterval.AsTimeDuration())
	p.Viper.SetDefault(NodeComputeManifestCache, cfg.Node.Compute.ManifestCache)
	p.Viper.SetDefault(NodeComputeManifestCacheSize, cfg.Node.Compute.ManifestCache.Size)
	p.Viper.SetDefault(NodeComputeManifestCacheDuration, cfg.Node.Compute.ManifestCache.Duration.AsTimeDuration())
	p.Viper.SetDefault(NodeComputeManifestCacheFrequency, cfg.Node.Compute.ManifestCache.Frequency.AsTimeDuration())
	p.Viper.SetDefault(NodeComputeLogStreamConfig, cfg.Node.Compute.LogStreamConfig)
	p.Viper.SetDefault(NodeComputeLogStreamConfigChannelBufferSize, cfg.Node.Compute.LogStreamConfig.ChannelBufferSize)
	p.Viper.SetDefault(NodeComputeLocalPublisher, cfg.Node.Compute.LocalPublisher)
	p.Viper.SetDefault(NodeComputeLocalPublisherAddress, cfg.Node.Compute.LocalPublisher.Address)
	p.Viper.SetDefault(NodeComputeLocalPublisherPort, cfg.Node.Compute.LocalPublisher.Port)
	p.Viper.SetDefault(NodeComputeLocalPublisherDirectory, cfg.Node.Compute.LocalPublisher.Directory)
	p.Viper.SetDefault(NodeComputeControlPlaneSettings, cfg.Node.Compute.ControlPlaneSettings)
	p.Viper.SetDefault(NodeComputeControlPlaneSettingsInfoUpdateFrequency, cfg.Node.Compute.ControlPlaneSettings.InfoUpdateFrequency.AsTimeDuration())
	p.Viper.SetDefault(NodeComputeControlPlaneSettingsResourceUpdateFrequency, cfg.Node.Compute.ControlPlaneSettings.ResourceUpdateFrequency.AsTimeDuration())
	p.Viper.SetDefault(NodeComputeControlPlaneSettingsHeartbeatFrequency, cfg.Node.Compute.ControlPlaneSettings.HeartbeatFrequency.AsTimeDuration())
	p.Viper.SetDefault(NodeComputeControlPlaneSettingsHeartbeatTopic, cfg.Node.Compute.ControlPlaneSettings.HeartbeatTopic)
	p.Viper.SetDefault(NodeRequester, cfg.Node.Requester)
	p.Viper.SetDefault(NodeRequesterJobDefaults, cfg.Node.Requester.JobDefaults)
	p.Viper.SetDefault(NodeRequesterJobDefaultsTotalTimeout, cfg.Node.Requester.JobDefaults.TotalTimeout.AsTimeDuration())
	p.Viper.SetDefault(NodeRequesterJobDefaultsExecutionTimeout, cfg.Node.Requester.JobDefaults.ExecutionTimeout.AsTimeDuration())
	p.Viper.SetDefault(NodeRequesterJobDefaultsQueueTimeout, cfg.Node.Requester.JobDefaults.QueueTimeout.AsTimeDuration())
	p.Viper.SetDefault(NodeRequesterExternalVerifierHook, cfg.Node.Requester.ExternalVerifierHook)
	p.Viper.SetDefault(NodeRequesterJobSelectionPolicy, cfg.Node.Requester.JobSelectionPolicy)
	p.Viper.SetDefault(NodeRequesterJobSelectionPolicyLocality, cfg.Node.Requester.JobSelectionPolicy.Locality)
	p.Viper.SetDefault(NodeRequesterJobSelectionPolicyRejectStatelessJobs, cfg.Node.Requester.JobSelectionPolicy.RejectStatelessJobs)
	p.Viper.SetDefault(NodeRequesterJobSelectionPolicyAcceptNetworkedJobs, cfg.Node.Requester.JobSelectionPolicy.AcceptNetworkedJobs)
	p.Viper.SetDefault(NodeRequesterJobSelectionPolicyProbeHTTP, cfg.Node.Requester.JobSelectionPolicy.ProbeHTTP)
	p.Viper.SetDefault(NodeRequesterJobSelectionPolicyProbeExec, cfg.Node.Requester.JobSelectionPolicy.ProbeExec)
	p.Viper.SetDefault(NodeRequesterJobStore, cfg.Node.Requester.JobStore)
	p.Viper.SetDefault(NodeRequesterJobStoreType, cfg.Node.Requester.JobStore.Type)
	p.Viper.SetDefault(NodeRequesterJobStorePath, cfg.Node.Requester.JobStore.Path)
	p.Viper.SetDefault(NodeRequesterHousekeepingBackgroundTaskInterval, cfg.Node.Requester.HousekeepingBackgroundTaskInterval.AsTimeDuration())
	p.Viper.SetDefault(NodeRequesterNodeRankRandomnessRange, cfg.Node.Requester.NodeRankRandomnessRange)
	p.Viper.SetDefault(NodeRequesterOverAskForBidsFactor, cfg.Node.Requester.OverAskForBidsFactor)
	p.Viper.SetDefault(NodeRequesterFailureInjectionConfig, cfg.Node.Requester.FailureInjectionConfig)
	p.Viper.SetDefault(NodeRequesterFailureInjectionConfigIsBadActor, cfg.Node.Requester.FailureInjectionConfig.IsBadActor)
	p.Viper.SetDefault(NodeRequesterTranslationEnabled, cfg.Node.Requester.TranslationEnabled)
	p.Viper.SetDefault(NodeRequesterEvaluationBroker, cfg.Node.Requester.EvaluationBroker)
	p.Viper.SetDefault(NodeRequesterEvaluationBrokerEvalBrokerVisibilityTimeout, cfg.Node.Requester.EvaluationBroker.EvalBrokerVisibilityTimeout.AsTimeDuration())
	p.Viper.SetDefault(NodeRequesterEvaluationBrokerEvalBrokerInitialRetryDelay, cfg.Node.Requester.EvaluationBroker.EvalBrokerInitialRetryDelay.AsTimeDuration())
	p.Viper.SetDefault(NodeRequesterEvaluationBrokerEvalBrokerSubsequentRetryDelay, cfg.Node.Requester.EvaluationBroker.EvalBrokerSubsequentRetryDelay.AsTimeDuration())
	p.Viper.SetDefault(NodeRequesterEvaluationBrokerEvalBrokerMaxRetryCount, cfg.Node.Requester.EvaluationBroker.EvalBrokerMaxRetryCount)
	p.Viper.SetDefault(NodeRequesterWorker, cfg.Node.Requester.Worker)
	p.Viper.SetDefault(NodeRequesterWorkerWorkerCount, cfg.Node.Requester.Worker.WorkerCount)
	p.Viper.SetDefault(NodeRequesterWorkerWorkerEvalDequeueTimeout, cfg.Node.Requester.Worker.WorkerEvalDequeueTimeout.AsTimeDuration())
	p.Viper.SetDefault(NodeRequesterWorkerWorkerEvalDequeueBaseBackoff, cfg.Node.Requester.Worker.WorkerEvalDequeueBaseBackoff.AsTimeDuration())
	p.Viper.SetDefault(NodeRequesterWorkerWorkerEvalDequeueMaxBackoff, cfg.Node.Requester.Worker.WorkerEvalDequeueMaxBackoff.AsTimeDuration())
	p.Viper.SetDefault(NodeRequesterScheduler, cfg.Node.Requester.Scheduler)
	p.Viper.SetDefault(NodeRequesterSchedulerQueueBackoff, cfg.Node.Requester.Scheduler.QueueBackoff.AsTimeDuration())
	p.Viper.SetDefault(NodeRequesterSchedulerNodeOverSubscriptionFactor, cfg.Node.Requester.Scheduler.NodeOverSubscriptionFactor)
	p.Viper.SetDefault(NodeRequesterStorageProvider, cfg.Node.Requester.StorageProvider)
	p.Viper.SetDefault(NodeRequesterStorageProviderS3, cfg.Node.Requester.StorageProvider.S3)
	p.Viper.SetDefault(NodeRequesterStorageProviderS3PreSignedURLDisabled, cfg.Node.Requester.StorageProvider.S3.PreSignedURLDisabled)
	p.Viper.SetDefault(NodeRequesterStorageProviderS3PreSignedURLExpiration, cfg.Node.Requester.StorageProvider.S3.PreSignedURLExpiration.AsTimeDuration())
	p.Viper.SetDefault(NodeRequesterTagCache, cfg.Node.Requester.TagCache)
	p.Viper.SetDefault(NodeRequesterTagCacheSize, cfg.Node.Requester.TagCache.Size)
	p.Viper.SetDefault(NodeRequesterTagCacheDuration, cfg.Node.Requester.TagCache.Duration.AsTimeDuration())
	p.Viper.SetDefault(NodeRequesterTagCacheFrequency, cfg.Node.Requester.TagCache.Frequency.AsTimeDuration())
	p.Viper.SetDefault(NodeRequesterDefaultPublisher, cfg.Node.Requester.DefaultPublisher)
	p.Viper.SetDefault(NodeRequesterControlPlaneSettings, cfg.Node.Requester.ControlPlaneSettings)
	p.Viper.SetDefault(NodeRequesterControlPlaneSettingsHeartbeatCheckFrequency, cfg.Node.Requester.ControlPlaneSettings.HeartbeatCheckFrequency.AsTimeDuration())
	p.Viper.SetDefault(NodeRequesterControlPlaneSettingsHeartbeatTopic, cfg.Node.Requester.ControlPlaneSettings.HeartbeatTopic)
	p.Viper.SetDefault(NodeRequesterControlPlaneSettingsNodeDisconnectedAfter, cfg.Node.Requester.ControlPlaneSettings.NodeDisconnectedAfter.AsTimeDuration())
	p.Viper.SetDefault(NodeRequesterNodeInfoStoreTTL, cfg.Node.Requester.NodeInfoStoreTTL.AsTimeDuration())
	p.Viper.SetDefault(NodeRequesterManualNodeApproval, cfg.Node.Requester.ManualNodeApproval)
	p.Viper.SetDefault(NodeDownloadURLRequestRetries, cfg.Node.DownloadURLRequestRetries)
	p.Viper.SetDefault(NodeDownloadURLRequestTimeout, cfg.Node.DownloadURLRequestTimeout.AsTimeDuration())
	p.Viper.SetDefault(NodeVolumeSizeRequestTimeout, cfg.Node.VolumeSizeRequestTimeout.AsTimeDuration())
	p.Viper.SetDefault(NodeExecutorPluginPath, cfg.Node.ExecutorPluginPath)
	p.Viper.SetDefault(NodeComputeStoragePath, cfg.Node.ComputeStoragePath)
	p.Viper.SetDefault(NodeLoggingMode, cfg.Node.LoggingMode)
	p.Viper.SetDefault(NodeType, cfg.Node.Type)
	p.Viper.SetDefault(NodeAllowListedLocalPaths, cfg.Node.AllowListedLocalPaths)
	p.Viper.SetDefault(NodeDisabledFeatures, cfg.Node.DisabledFeatures)
	p.Viper.SetDefault(NodeDisabledFeaturesEngines, cfg.Node.DisabledFeatures.Engines)
	p.Viper.SetDefault(NodeDisabledFeaturesPublishers, cfg.Node.DisabledFeatures.Publishers)
	p.Viper.SetDefault(NodeDisabledFeaturesStorages, cfg.Node.DisabledFeatures.Storages)
	p.Viper.SetDefault(NodeLabels, cfg.Node.Labels)
	p.Viper.SetDefault(NodeWebUI, cfg.Node.WebUI)
	p.Viper.SetDefault(NodeWebUIEnabled, cfg.Node.WebUI.Enabled)
	p.Viper.SetDefault(NodeWebUIPort, cfg.Node.WebUI.Port)
	p.Viper.SetDefault(NodeNetwork, cfg.Node.Network)
	p.Viper.SetDefault(NodeNetworkPort, cfg.Node.Network.Port)
	p.Viper.SetDefault(NodeNetworkAdvertisedAddress, cfg.Node.Network.AdvertisedAddress)
	p.Viper.SetDefault(NodeNetworkAuthSecret, cfg.Node.Network.AuthSecret)
	p.Viper.SetDefault(NodeNetworkOrchestrators, cfg.Node.Network.Orchestrators)
	p.Viper.SetDefault(NodeNetworkStoreDir, cfg.Node.Network.StoreDir)
	p.Viper.SetDefault(NodeNetworkCluster, cfg.Node.Network.Cluster)
	p.Viper.SetDefault(NodeNetworkClusterName, cfg.Node.Network.Cluster.Name)
	p.Viper.SetDefault(NodeNetworkClusterPort, cfg.Node.Network.Cluster.Port)
	p.Viper.SetDefault(NodeNetworkClusterAdvertisedAddress, cfg.Node.Network.Cluster.AdvertisedAddress)
	p.Viper.SetDefault(NodeNetworkClusterPeers, cfg.Node.Network.Cluster.Peers)
	p.Viper.SetDefault(NodeStrictVersionMatch, cfg.Node.StrictVersionMatch)
	p.Viper.SetDefault(User, cfg.User)
	p.Viper.SetDefault(UserKeyPath, cfg.User.KeyPath)
	p.Viper.SetDefault(UserInstallationID, cfg.User.InstallationID)
	p.Viper.SetDefault(Metrics, cfg.Metrics)
	p.Viper.SetDefault(MetricsEventTracerPath, cfg.Metrics.EventTracerPath)
	p.Viper.SetDefault(Update, cfg.Update)
	p.Viper.SetDefault(UpdateSkipChecks, cfg.Update.SkipChecks)
	p.Viper.SetDefault(UpdateCheckFrequency, cfg.Update.CheckFrequency.AsTimeDuration())
	p.Viper.SetDefault(Auth, cfg.Auth)
	p.Viper.SetDefault(AuthTokensPath, cfg.Auth.TokensPath)
	p.Viper.SetDefault(AuthMethods, cfg.Auth.Methods)
	p.Viper.SetDefault(AuthAccessPolicyPath, cfg.Auth.AccessPolicyPath)

}

func Set(cfg BacalhauConfig, opts ...SetOption) {
	p := &SetParams{
		Viper: viper.GetViper(),
	}
	for _, opt := range opts {
		opt(p)
	}

	p.Viper.Set(Node, cfg.Node)
	p.Viper.Set(NodeName, cfg.Node.Name)
	p.Viper.Set(NodeNameProvider, cfg.Node.NameProvider)
	p.Viper.Set(NodeClientAPI, cfg.Node.ClientAPI)
	p.Viper.Set(NodeClientAPIHost, cfg.Node.ClientAPI.Host)
	p.Viper.Set(NodeClientAPIPort, cfg.Node.ClientAPI.Port)
	p.Viper.Set(NodeClientAPIClientTLS, cfg.Node.ClientAPI.ClientTLS)
	p.Viper.Set(NodeClientAPIClientTLSUseTLS, cfg.Node.ClientAPI.ClientTLS.UseTLS)
	p.Viper.Set(NodeClientAPIClientTLSCACert, cfg.Node.ClientAPI.ClientTLS.CACert)
	p.Viper.Set(NodeClientAPIClientTLSInsecure, cfg.Node.ClientAPI.ClientTLS.Insecure)
	p.Viper.Set(NodeClientAPITLS, cfg.Node.ClientAPI.TLS)
	p.Viper.Set(NodeClientAPITLSAutoCert, cfg.Node.ClientAPI.TLS.AutoCert)
	p.Viper.Set(NodeClientAPITLSAutoCertCachePath, cfg.Node.ClientAPI.TLS.AutoCertCachePath)
	p.Viper.Set(NodeClientAPITLSServerCertificate, cfg.Node.ClientAPI.TLS.ServerCertificate)
	p.Viper.Set(NodeClientAPITLSServerKey, cfg.Node.ClientAPI.TLS.ServerKey)
	p.Viper.Set(NodeClientAPITLSSelfSigned, cfg.Node.ClientAPI.TLS.SelfSigned)
	p.Viper.Set(NodeServerAPI, cfg.Node.ServerAPI)
	p.Viper.Set(NodeServerAPIHost, cfg.Node.ServerAPI.Host)
	p.Viper.Set(NodeServerAPIPort, cfg.Node.ServerAPI.Port)
	p.Viper.Set(NodeServerAPIClientTLS, cfg.Node.ServerAPI.ClientTLS)
	p.Viper.Set(NodeServerAPIClientTLSUseTLS, cfg.Node.ServerAPI.ClientTLS.UseTLS)
	p.Viper.Set(NodeServerAPIClientTLSCACert, cfg.Node.ServerAPI.ClientTLS.CACert)
	p.Viper.Set(NodeServerAPIClientTLSInsecure, cfg.Node.ServerAPI.ClientTLS.Insecure)
	p.Viper.Set(NodeServerAPITLS, cfg.Node.ServerAPI.TLS)
	p.Viper.Set(NodeServerAPITLSAutoCert, cfg.Node.ServerAPI.TLS.AutoCert)
	p.Viper.Set(NodeServerAPITLSAutoCertCachePath, cfg.Node.ServerAPI.TLS.AutoCertCachePath)
	p.Viper.Set(NodeServerAPITLSServerCertificate, cfg.Node.ServerAPI.TLS.ServerCertificate)
	p.Viper.Set(NodeServerAPITLSServerKey, cfg.Node.ServerAPI.TLS.ServerKey)
	p.Viper.Set(NodeServerAPITLSSelfSigned, cfg.Node.ServerAPI.TLS.SelfSigned)
	p.Viper.Set(NodeIPFS, cfg.Node.IPFS)
	p.Viper.Set(NodeIPFSConnect, cfg.Node.IPFS.Connect)
	p.Viper.Set(NodeCompute, cfg.Node.Compute)
	p.Viper.Set(NodeComputeCapacity, cfg.Node.Compute.Capacity)
	p.Viper.Set(NodeComputeCapacityIgnorePhysicalResourceLimits, cfg.Node.Compute.Capacity.IgnorePhysicalResourceLimits)
	p.Viper.Set(NodeComputeCapacityTotalResourceLimits, cfg.Node.Compute.Capacity.TotalResourceLimits)
	p.Viper.Set(NodeComputeCapacityTotalResourceLimitsCPU, cfg.Node.Compute.Capacity.TotalResourceLimits.CPU)
	p.Viper.Set(NodeComputeCapacityTotalResourceLimitsMemory, cfg.Node.Compute.Capacity.TotalResourceLimits.Memory)
	p.Viper.Set(NodeComputeCapacityTotalResourceLimitsDisk, cfg.Node.Compute.Capacity.TotalResourceLimits.Disk)
	p.Viper.Set(NodeComputeCapacityTotalResourceLimitsGPU, cfg.Node.Compute.Capacity.TotalResourceLimits.GPU)
	p.Viper.Set(NodeComputeCapacityJobResourceLimits, cfg.Node.Compute.Capacity.JobResourceLimits)
	p.Viper.Set(NodeComputeCapacityJobResourceLimitsCPU, cfg.Node.Compute.Capacity.JobResourceLimits.CPU)
	p.Viper.Set(NodeComputeCapacityJobResourceLimitsMemory, cfg.Node.Compute.Capacity.JobResourceLimits.Memory)
	p.Viper.Set(NodeComputeCapacityJobResourceLimitsDisk, cfg.Node.Compute.Capacity.JobResourceLimits.Disk)
	p.Viper.Set(NodeComputeCapacityJobResourceLimitsGPU, cfg.Node.Compute.Capacity.JobResourceLimits.GPU)
	p.Viper.Set(NodeComputeCapacityDefaultJobResourceLimits, cfg.Node.Compute.Capacity.DefaultJobResourceLimits)
	p.Viper.Set(NodeComputeCapacityDefaultJobResourceLimitsCPU, cfg.Node.Compute.Capacity.DefaultJobResourceLimits.CPU)
	p.Viper.Set(NodeComputeCapacityDefaultJobResourceLimitsMemory, cfg.Node.Compute.Capacity.DefaultJobResourceLimits.Memory)
	p.Viper.Set(NodeComputeCapacityDefaultJobResourceLimitsDisk, cfg.Node.Compute.Capacity.DefaultJobResourceLimits.Disk)
	p.Viper.Set(NodeComputeCapacityDefaultJobResourceLimitsGPU, cfg.Node.Compute.Capacity.DefaultJobResourceLimits.GPU)
	p.Viper.Set(NodeComputeExecutionStore, cfg.Node.Compute.ExecutionStore)
	p.Viper.Set(NodeComputeExecutionStoreType, cfg.Node.Compute.ExecutionStore.Type)
	p.Viper.Set(NodeComputeExecutionStorePath, cfg.Node.Compute.ExecutionStore.Path)
	p.Viper.Set(NodeComputeJobTimeouts, cfg.Node.Compute.JobTimeouts)
	p.Viper.Set(NodeComputeJobTimeoutsJobExecutionTimeoutClientIDBypassList, cfg.Node.Compute.JobTimeouts.JobExecutionTimeoutClientIDBypassList)
	p.Viper.Set(NodeComputeJobTimeoutsJobNegotiationTimeout, cfg.Node.Compute.JobTimeouts.JobNegotiationTimeout.AsTimeDuration())
	p.Viper.Set(NodeComputeJobTimeoutsMinJobExecutionTimeout, cfg.Node.Compute.JobTimeouts.MinJobExecutionTimeout.AsTimeDuration())
	p.Viper.Set(NodeComputeJobTimeoutsMaxJobExecutionTimeout, cfg.Node.Compute.JobTimeouts.MaxJobExecutionTimeout.AsTimeDuration())
	p.Viper.Set(NodeComputeJobTimeoutsDefaultJobExecutionTimeout, cfg.Node.Compute.JobTimeouts.DefaultJobExecutionTimeout.AsTimeDuration())
	p.Viper.Set(NodeComputeJobSelection, cfg.Node.Compute.JobSelection)
	p.Viper.Set(NodeComputeJobSelectionLocality, cfg.Node.Compute.JobSelection.Locality)
	p.Viper.Set(NodeComputeJobSelectionRejectStatelessJobs, cfg.Node.Compute.JobSelection.RejectStatelessJobs)
	p.Viper.Set(NodeComputeJobSelectionAcceptNetworkedJobs, cfg.Node.Compute.JobSelection.AcceptNetworkedJobs)
	p.Viper.Set(NodeComputeJobSelectionProbeHTTP, cfg.Node.Compute.JobSelection.ProbeHTTP)
	p.Viper.Set(NodeComputeJobSelectionProbeExec, cfg.Node.Compute.JobSelection.ProbeExec)
	p.Viper.Set(NodeComputeLogging, cfg.Node.Compute.Logging)
	p.Viper.Set(NodeComputeLoggingLogRunningExecutionsInterval, cfg.Node.Compute.Logging.LogRunningExecutionsInterval.AsTimeDuration())
	p.Viper.Set(NodeComputeManifestCache, cfg.Node.Compute.ManifestCache)
	p.Viper.Set(NodeComputeManifestCacheSize, cfg.Node.Compute.ManifestCache.Size)
	p.Viper.Set(NodeComputeManifestCacheDuration, cfg.Node.Compute.ManifestCache.Duration.AsTimeDuration())
	p.Viper.Set(NodeComputeManifestCacheFrequency, cfg.Node.Compute.ManifestCache.Frequency.AsTimeDuration())
	p.Viper.Set(NodeComputeLogStreamConfig, cfg.Node.Compute.LogStreamConfig)
	p.Viper.Set(NodeComputeLogStreamConfigChannelBufferSize, cfg.Node.Compute.LogStreamConfig.ChannelBufferSize)
	p.Viper.Set(NodeComputeLocalPublisher, cfg.Node.Compute.LocalPublisher)
	p.Viper.Set(NodeComputeLocalPublisherAddress, cfg.Node.Compute.LocalPublisher.Address)
	p.Viper.Set(NodeComputeLocalPublisherPort, cfg.Node.Compute.LocalPublisher.Port)
	p.Viper.Set(NodeComputeLocalPublisherDirectory, cfg.Node.Compute.LocalPublisher.Directory)
	p.Viper.Set(NodeComputeControlPlaneSettings, cfg.Node.Compute.ControlPlaneSettings)
	p.Viper.Set(NodeComputeControlPlaneSettingsInfoUpdateFrequency, cfg.Node.Compute.ControlPlaneSettings.InfoUpdateFrequency.AsTimeDuration())
	p.Viper.Set(NodeComputeControlPlaneSettingsResourceUpdateFrequency, cfg.Node.Compute.ControlPlaneSettings.ResourceUpdateFrequency.AsTimeDuration())
	p.Viper.Set(NodeComputeControlPlaneSettingsHeartbeatFrequency, cfg.Node.Compute.ControlPlaneSettings.HeartbeatFrequency.AsTimeDuration())
	p.Viper.Set(NodeComputeControlPlaneSettingsHeartbeatTopic, cfg.Node.Compute.ControlPlaneSettings.HeartbeatTopic)
	p.Viper.Set(NodeRequester, cfg.Node.Requester)
	p.Viper.Set(NodeRequesterJobDefaults, cfg.Node.Requester.JobDefaults)
	p.Viper.Set(NodeRequesterJobDefaultsTotalTimeout, cfg.Node.Requester.JobDefaults.TotalTimeout.AsTimeDuration())
	p.Viper.Set(NodeRequesterJobDefaultsExecutionTimeout, cfg.Node.Requester.JobDefaults.ExecutionTimeout.AsTimeDuration())
	p.Viper.Set(NodeRequesterJobDefaultsQueueTimeout, cfg.Node.Requester.JobDefaults.QueueTimeout.AsTimeDuration())
	p.Viper.Set(NodeRequesterExternalVerifierHook, cfg.Node.Requester.ExternalVerifierHook)
	p.Viper.Set(NodeRequesterJobSelectionPolicy, cfg.Node.Requester.JobSelectionPolicy)
	p.Viper.Set(NodeRequesterJobSelectionPolicyLocality, cfg.Node.Requester.JobSelectionPolicy.Locality)
	p.Viper.Set(NodeRequesterJobSelectionPolicyRejectStatelessJobs, cfg.Node.Requester.JobSelectionPolicy.RejectStatelessJobs)
	p.Viper.Set(NodeRequesterJobSelectionPolicyAcceptNetworkedJobs, cfg.Node.Requester.JobSelectionPolicy.AcceptNetworkedJobs)
	p.Viper.Set(NodeRequesterJobSelectionPolicyProbeHTTP, cfg.Node.Requester.JobSelectionPolicy.ProbeHTTP)
	p.Viper.Set(NodeRequesterJobSelectionPolicyProbeExec, cfg.Node.Requester.JobSelectionPolicy.ProbeExec)
	p.Viper.Set(NodeRequesterJobStore, cfg.Node.Requester.JobStore)
	p.Viper.Set(NodeRequesterJobStoreType, cfg.Node.Requester.JobStore.Type)
	p.Viper.Set(NodeRequesterJobStorePath, cfg.Node.Requester.JobStore.Path)
	p.Viper.Set(NodeRequesterHousekeepingBackgroundTaskInterval, cfg.Node.Requester.HousekeepingBackgroundTaskInterval.AsTimeDuration())
	p.Viper.Set(NodeRequesterNodeRankRandomnessRange, cfg.Node.Requester.NodeRankRandomnessRange)
	p.Viper.Set(NodeRequesterOverAskForBidsFactor, cfg.Node.Requester.OverAskForBidsFactor)
	p.Viper.Set(NodeRequesterFailureInjectionConfig, cfg.Node.Requester.FailureInjectionConfig)
	p.Viper.Set(NodeRequesterFailureInjectionConfigIsBadActor, cfg.Node.Requester.FailureInjectionConfig.IsBadActor)
	p.Viper.Set(NodeRequesterTranslationEnabled, cfg.Node.Requester.TranslationEnabled)
	p.Viper.Set(NodeRequesterEvaluationBroker, cfg.Node.Requester.EvaluationBroker)
	p.Viper.Set(NodeRequesterEvaluationBrokerEvalBrokerVisibilityTimeout, cfg.Node.Requester.EvaluationBroker.EvalBrokerVisibilityTimeout.AsTimeDuration())
	p.Viper.Set(NodeRequesterEvaluationBrokerEvalBrokerInitialRetryDelay, cfg.Node.Requester.EvaluationBroker.EvalBrokerInitialRetryDelay.AsTimeDuration())
	p.Viper.Set(NodeRequesterEvaluationBrokerEvalBrokerSubsequentRetryDelay, cfg.Node.Requester.EvaluationBroker.EvalBrokerSubsequentRetryDelay.AsTimeDuration())
	p.Viper.Set(NodeRequesterEvaluationBrokerEvalBrokerMaxRetryCount, cfg.Node.Requester.EvaluationBroker.EvalBrokerMaxRetryCount)
	p.Viper.Set(NodeRequesterWorker, cfg.Node.Requester.Worker)
	p.Viper.Set(NodeRequesterWorkerWorkerCount, cfg.Node.Requester.Worker.WorkerCount)
	p.Viper.Set(NodeRequesterWorkerWorkerEvalDequeueTimeout, cfg.Node.Requester.Worker.WorkerEvalDequeueTimeout.AsTimeDuration())
	p.Viper.Set(NodeRequesterWorkerWorkerEvalDequeueBaseBackoff, cfg.Node.Requester.Worker.WorkerEvalDequeueBaseBackoff.AsTimeDuration())
	p.Viper.Set(NodeRequesterWorkerWorkerEvalDequeueMaxBackoff, cfg.Node.Requester.Worker.WorkerEvalDequeueMaxBackoff.AsTimeDuration())
	p.Viper.Set(NodeRequesterScheduler, cfg.Node.Requester.Scheduler)
	p.Viper.Set(NodeRequesterSchedulerQueueBackoff, cfg.Node.Requester.Scheduler.QueueBackoff.AsTimeDuration())
	p.Viper.Set(NodeRequesterSchedulerNodeOverSubscriptionFactor, cfg.Node.Requester.Scheduler.NodeOverSubscriptionFactor)
	p.Viper.Set(NodeRequesterStorageProvider, cfg.Node.Requester.StorageProvider)
	p.Viper.Set(NodeRequesterStorageProviderS3, cfg.Node.Requester.StorageProvider.S3)
	p.Viper.Set(NodeRequesterStorageProviderS3PreSignedURLDisabled, cfg.Node.Requester.StorageProvider.S3.PreSignedURLDisabled)
	p.Viper.Set(NodeRequesterStorageProviderS3PreSignedURLExpiration, cfg.Node.Requester.StorageProvider.S3.PreSignedURLExpiration.AsTimeDuration())
	p.Viper.Set(NodeRequesterTagCache, cfg.Node.Requester.TagCache)
	p.Viper.Set(NodeRequesterTagCacheSize, cfg.Node.Requester.TagCache.Size)
	p.Viper.Set(NodeRequesterTagCacheDuration, cfg.Node.Requester.TagCache.Duration.AsTimeDuration())
	p.Viper.Set(NodeRequesterTagCacheFrequency, cfg.Node.Requester.TagCache.Frequency.AsTimeDuration())
	p.Viper.Set(NodeRequesterDefaultPublisher, cfg.Node.Requester.DefaultPublisher)
	p.Viper.Set(NodeRequesterControlPlaneSettings, cfg.Node.Requester.ControlPlaneSettings)
	p.Viper.Set(NodeRequesterControlPlaneSettingsHeartbeatCheckFrequency, cfg.Node.Requester.ControlPlaneSettings.HeartbeatCheckFrequency.AsTimeDuration())
	p.Viper.Set(NodeRequesterControlPlaneSettingsHeartbeatTopic, cfg.Node.Requester.ControlPlaneSettings.HeartbeatTopic)
	p.Viper.Set(NodeRequesterControlPlaneSettingsNodeDisconnectedAfter, cfg.Node.Requester.ControlPlaneSettings.NodeDisconnectedAfter.AsTimeDuration())
	p.Viper.Set(NodeRequesterNodeInfoStoreTTL, cfg.Node.Requester.NodeInfoStoreTTL.AsTimeDuration())
	p.Viper.Set(NodeRequesterManualNodeApproval, cfg.Node.Requester.ManualNodeApproval)
	p.Viper.Set(NodeDownloadURLRequestRetries, cfg.Node.DownloadURLRequestRetries)
	p.Viper.Set(NodeDownloadURLRequestTimeout, cfg.Node.DownloadURLRequestTimeout.AsTimeDuration())
	p.Viper.Set(NodeVolumeSizeRequestTimeout, cfg.Node.VolumeSizeRequestTimeout.AsTimeDuration())
	p.Viper.Set(NodeExecutorPluginPath, cfg.Node.ExecutorPluginPath)
	p.Viper.Set(NodeComputeStoragePath, cfg.Node.ComputeStoragePath)
	p.Viper.Set(NodeLoggingMode, cfg.Node.LoggingMode)
	p.Viper.Set(NodeType, cfg.Node.Type)
	p.Viper.Set(NodeAllowListedLocalPaths, cfg.Node.AllowListedLocalPaths)
	p.Viper.Set(NodeDisabledFeatures, cfg.Node.DisabledFeatures)
	p.Viper.Set(NodeDisabledFeaturesEngines, cfg.Node.DisabledFeatures.Engines)
	p.Viper.Set(NodeDisabledFeaturesPublishers, cfg.Node.DisabledFeatures.Publishers)
	p.Viper.Set(NodeDisabledFeaturesStorages, cfg.Node.DisabledFeatures.Storages)
	p.Viper.Set(NodeLabels, cfg.Node.Labels)
	p.Viper.Set(NodeWebUI, cfg.Node.WebUI)
	p.Viper.Set(NodeWebUIEnabled, cfg.Node.WebUI.Enabled)
	p.Viper.Set(NodeWebUIPort, cfg.Node.WebUI.Port)
	p.Viper.Set(NodeNetwork, cfg.Node.Network)
	p.Viper.Set(NodeNetworkPort, cfg.Node.Network.Port)
	p.Viper.Set(NodeNetworkAdvertisedAddress, cfg.Node.Network.AdvertisedAddress)
	p.Viper.Set(NodeNetworkAuthSecret, cfg.Node.Network.AuthSecret)
	p.Viper.Set(NodeNetworkOrchestrators, cfg.Node.Network.Orchestrators)
	p.Viper.Set(NodeNetworkStoreDir, cfg.Node.Network.StoreDir)
	p.Viper.Set(NodeNetworkCluster, cfg.Node.Network.Cluster)
	p.Viper.Set(NodeNetworkClusterName, cfg.Node.Network.Cluster.Name)
	p.Viper.Set(NodeNetworkClusterPort, cfg.Node.Network.Cluster.Port)
	p.Viper.Set(NodeNetworkClusterAdvertisedAddress, cfg.Node.Network.Cluster.AdvertisedAddress)
	p.Viper.Set(NodeNetworkClusterPeers, cfg.Node.Network.Cluster.Peers)
	p.Viper.Set(NodeStrictVersionMatch, cfg.Node.StrictVersionMatch)
	p.Viper.Set(User, cfg.User)
	p.Viper.Set(UserKeyPath, cfg.User.KeyPath)
	p.Viper.Set(UserInstallationID, cfg.User.InstallationID)
	p.Viper.Set(Metrics, cfg.Metrics)
	p.Viper.Set(MetricsEventTracerPath, cfg.Metrics.EventTracerPath)
	p.Viper.Set(Update, cfg.Update)
	p.Viper.Set(UpdateSkipChecks, cfg.Update.SkipChecks)
	p.Viper.Set(UpdateCheckFrequency, cfg.Update.CheckFrequency.AsTimeDuration())
	p.Viper.Set(Auth, cfg.Auth)
	p.Viper.Set(AuthTokensPath, cfg.Auth.TokensPath)
	p.Viper.Set(AuthMethods, cfg.Auth.Methods)
	p.Viper.Set(AuthAccessPolicyPath, cfg.Auth.AccessPolicyPath)
}

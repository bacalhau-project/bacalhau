// Code generated by go generate; DO NOT EDIT.

package types

// ConfigDescriptions maps configuration paths to their descriptions
var ConfigDescriptions = map[string]string{
	APIHostKey:                                       "Host specifies the hostname or IP address on which the API server listens or the client connects.",
	APIPortKey:                                       "Port specifies the port number on which the API server listens or the client connects.",
	APITLSCertFileKey:                                "CertFile specifies the path to the TLS certificate file.",
	APITLSKeyFileKey:                                 "KeyFile specifies the path to the TLS private key file.",
	APITLSCAFileKey:                                  "CAFile specifies the path to the Certificate Authority file.",
	APITLSUseTLSKey:                                  "UseTLS indicates whether to use TLS for client connections.",
	APITLSInsecureKey:                                "Insecure allows insecure TLS connections (e.g., self-signed certificates).",
	APITLSSelfSignedKey:                              "SelfSigned indicates whether to use a self-signed certificate.",
	APITLSAutoCertKey:                                "AutoCert specifies the domain for automatic certificate generation.",
	APITLSAutoCertCachePathKey:                       "AutoCertCachePath specifies the directory to cache auto-generated certificates.",
	APIAuthMethodsKey:                                "Methods maps \"method names\" to authenticator implementations. A method name is a human-readable string chosen by the person configuring the system that is shown to users to help them pick the authentication method they want to use. There can be multiple usages of the same Authenticator *type* but with different configs and parameters, each identified with a unique method name.  For example, if an implementation wants to allow users to log in with Github or Bitbucket, they might both use an authenticator implementation of type \"oidc\", and each would appear once on this provider with key / method name \"github\" and \"bitbucket\".  By default, only a single authentication method that accepts authentication via client keys will be enabled.",
	APIAuthAccessPolicyPathKey:                       "AccessPolicyPath is the path to a file or directory that will be loaded as the policy to apply to all inbound API requests. If unspecified, a policy that permits access to all API endpoints to both authenticated and unauthenticated users (the default as of v1.2.0) will be used.",
	NameProviderKey:                                  "NameProvider specifies the method used to generate names for the node. One of: hostname, aws, gcp, uuid, puuid.",
	DataDirKey:                                       "DataDir specifies a location on disk where the bacalhau node will maintain state.",
	StrictVersionMatchKey:                            "StrictVersionMatch indicates whether to enforce strict version matching.",
	OrchestratorEnabledKey:                           "Enabled indicates whether the Web UI is enabled.",
	OrchestratorHostKey:                              "Host specifies the hostname or IP address on which the API server listens or the client connects.",
	OrchestratorPortKey:                              "Port specifies the port number on which the API server listens or the client connects.",
	OrchestratorAdvertiseKey:                         "Advertise specifies the address to advertise to other cluster members.",
	OrchestratorAuthSecretKey:                        "AuthSecret key specifies the key used by compute nodes to connect to an orchestrator.",
	OrchestratorTLSCertFileKey:                       "CertFile specifies the path to the TLS certificate file.",
	OrchestratorTLSKeyFileKey:                        "KeyFile specifies the path to the TLS private key file.",
	OrchestratorTLSCAFileKey:                         "CAFile specifies the path to the Certificate Authority file.",
	OrchestratorTLSUseTLSKey:                         "UseTLS indicates whether to use TLS for client connections.",
	OrchestratorTLSInsecureKey:                       "Insecure allows insecure TLS connections (e.g., self-signed certificates).",
	OrchestratorTLSSelfSignedKey:                     "SelfSigned indicates whether to use a self-signed certificate.",
	OrchestratorTLSAutoCertKey:                       "AutoCert specifies the domain for automatic certificate generation.",
	OrchestratorTLSAutoCertCachePathKey:              "AutoCertCachePath specifies the directory to cache auto-generated certificates.",
	OrchestratorClusterNameKey:                       "Name specifies the unique identifier for this orchestrator cluster.",
	OrchestratorClusterHostKey:                       "Host specifies the hostname or IP address on which the API server listens or the client connects.",
	OrchestratorClusterPortKey:                       "Port specifies the port number on which the API server listens or the client connects.",
	OrchestratorClusterAdvertiseKey:                  "Advertise specifies the address to advertise to other cluster members.",
	OrchestratorClusterPeersKey:                      "Peers is a list of other cluster members to connect to on startup.",
	OrchestratorNodeManagerDisconnectTimeoutKey:      "DisconnectTimeout specifies how long to wait before considering a node disconnected.",
	OrchestratorNodeManagerManualApprovalKey:         "ManualApproval, if true, requires manual approval for new compute nodes joining the cluster.",
	OrchestratorSchedulerWorkerCountKey:              "WorkerCount specifies the number of concurrent workers for job scheduling.",
	OrchestratorSchedulerHousekeepingIntervalKey:     "HousekeepingInterval specifies how often to run housekeeping tasks.",
	OrchestratorSchedulerHousekeepingTimeoutKey:      "HousekeepingTimeout specifies the maximum time allowed for a single housekeeping run.",
	OrchestratorEvaluationBrokerVisibilityTimeoutKey: "VisibilityTimeout specifies how long an evaluation can be claimed before it's returned to the queue.",
	OrchestratorEvaluationBrokerMaxRetryCountKey:     "ReadTimeout specifies the maximum number of attempts for reading from a storage.",
	ComputeEnabledKey:                                "Enabled indicates whether the Web UI is enabled.",
	ComputeOrchestratorsKey:                          "Orchestrators specifies a list of orchestrator endpoints that this compute node connects to.",
	ComputeTLSCertFileKey:                            "CertFile specifies the path to the TLS certificate file.",
	ComputeTLSKeyFileKey:                             "KeyFile specifies the path to the TLS private key file.",
	ComputeTLSCAFileKey:                              "CAFile specifies the path to the Certificate Authority file.",
	ComputeTLSUseTLSKey:                              "UseTLS indicates whether to use TLS for client connections.",
	ComputeTLSInsecureKey:                            "Insecure allows insecure TLS connections (e.g., self-signed certificates).",
	ComputeTLSSelfSignedKey:                          "SelfSigned indicates whether to use a self-signed certificate.",
	ComputeTLSAutoCertKey:                            "AutoCert specifies the domain for automatic certificate generation.",
	ComputeTLSAutoCertCachePathKey:                   "AutoCertCachePath specifies the directory to cache auto-generated certificates.",
	ComputeHeartbeatInfoUpdateIntervalKey:            "InfoUpdateInterval specifies the time between updates of non-resource information to the orchestrator.",
	ComputeHeartbeatResourceUpdateIntervalKey:        "ResourceUpdateInterval specifies the time between updates of resource information to the orchestrator.",
	ComputeHeartbeatIntervalKey:                      "Interval specifies the time between update checks, when set to 0 update checks are not performed.",
	ComputeLabelsKey:                                 "Labels are key-value pairs used to describe and categorize the compute node.",
	ComputeAllocatedCapacityCPUKey:                   "CPU specifies the default amount of CPU allocated to a task. It uses Kubernetes resource string format (e.g., \"100m\" for 0.1 CPU cores). This value is used when the task hasn't explicitly set its CPU requirement.",
	ComputeAllocatedCapacityMemoryKey:                "Memory specifies the default amount of memory allocated to a task. It uses Kubernetes resource string format (e.g., \"256Mi\" for 256 mebibytes). This value is used when the task hasn't explicitly set its memory requirement.",
	ComputeAllocatedCapacityDiskKey:                  "Disk specifies the default amount of disk space allocated to a task. It uses Kubernetes resource string format (e.g., \"1Gi\" for 1 gibibyte). This value is used when the task hasn't explicitly set its disk space requirement.",
	ComputeAllocatedCapacityGPUKey:                   "GPU specifies the default number of GPUs allocated to a task. It uses Kubernetes resource string format (e.g., \"1\" for 1 GPU). This value is used when the task hasn't explicitly set its GPU requirement.",
	ComputeAllowListedLocalPathsKey:                  "AllowListedLocalPaths specifies a list of local file system paths that the compute node is allowed to access.",
	WebUIEnabledKey:                                  "Enabled indicates whether the Web UI is enabled.",
	WebUIListenKey:                                   "Listen specifies the address and port on which the Web UI listens.",
	InputSourcesDisabledKey:                          "Disabled is a list of downloaders that are disabled.",
	InputSourcesReadTimeoutKey:                       "ReadTimeout specifies the maximum time allowed for reading from a storage.",
	InputSourcesMaxRetryCountKey:                     "ReadTimeout specifies the maximum number of attempts for reading from a storage.",
	InputSourcesTypesIPFSEndpointKey:                 "Endpoint specifies the multi-address to connect to for IPFS. e.g /ip4/127.0.0.1/tcp/5001",
	InputSourcesTypesS3EndpointKey:                   "Endpoint specifies the multi-address to connect to for IPFS. e.g /ip4/127.0.0.1/tcp/5001",
	InputSourcesTypesS3AccessKeyKey:                  "AccessKey specifies the access key for the S3 input source.",
	InputSourcesTypesS3SecretKeyKey:                  "SecretKey specifies the secret key for the S3 input source.",
	PublishersDisabledKey:                            "Disabled is a list of downloaders that are disabled.",
	PublishersTypesIPFSEndpointKey:                   "Endpoint specifies the multi-address to connect to for IPFS. e.g /ip4/127.0.0.1/tcp/5001",
	PublishersTypesS3PreSignedURLDisabledKey:         "PreSignedURLDisabled specifies whether pre-signed URLs are enabled for the S3 provider.",
	PublishersTypesS3PreSignedURLExpirationKey:       "PreSignedURLExpiration specifies the duration before a pre-signed URL expires.",
	PublishersTypesLocalAddressKey:                   "Address specifies the endpoint the publisher serves on.",
	PublishersTypesLocalPortKey:                      "Port specifies the port number on which the API server listens or the client connects.",
	PublishersTypesLocalDirectoryKey:                 "Directory specifies a path to location on disk where content is served from.",
	EnginesDisabledKey:                               "Disabled is a list of downloaders that are disabled.",
	EnginesTypesDockerManifestCacheSizeKey:           "Size specifies the size of the Docker manifest cache.",
	EnginesTypesDockerManifestCacheTTLKey:            "TTL specifies the time-to-live duration for cache entries.",
	EnginesTypesDockerManifestCacheRefreshKey:        "Refresh specifies the refresh interval for cache entries.",
	ResultDownloadersDisabledKey:                     "Disabled is a list of downloaders that are disabled.",
	ResultDownloadersTimeoutKey:                      "Timeout specifies the maximum time allowed for a download operation.",
	ResultDownloadersTypesIPFSEndpointKey:            "Endpoint specifies the multi-address to connect to for IPFS. e.g /ip4/127.0.0.1/tcp/5001",
	JobDefaultsBatchPriorityKey:                      "Priority specifies the default priority allocated to a service or daemon job. This value is used when the job hasn't explicitly set its priority requirement.",
	JobDefaultsBatchTaskResourcesCPUKey:              "CPU specifies the default amount of CPU allocated to a task. It uses Kubernetes resource string format (e.g., \"100m\" for 0.1 CPU cores). This value is used when the task hasn't explicitly set its CPU requirement.",
	JobDefaultsBatchTaskResourcesMemoryKey:           "Memory specifies the default amount of memory allocated to a task. It uses Kubernetes resource string format (e.g., \"256Mi\" for 256 mebibytes). This value is used when the task hasn't explicitly set its memory requirement.",
	JobDefaultsBatchTaskResourcesDiskKey:             "Disk specifies the default amount of disk space allocated to a task. It uses Kubernetes resource string format (e.g., \"1Gi\" for 1 gibibyte). This value is used when the task hasn't explicitly set its disk space requirement.",
	JobDefaultsBatchTaskResourcesGPUKey:              "GPU specifies the default number of GPUs allocated to a task. It uses Kubernetes resource string format (e.g., \"1\" for 1 GPU). This value is used when the task hasn't explicitly set its GPU requirement.",
	JobDefaultsBatchTaskPublisherConfigTypeKey:       "No description available",
	JobDefaultsBatchTaskPublisherConfigParamsKey:     "No description available",
	JobDefaultsBatchTaskTimeoutsTotalTimeoutKey:      "TotalTimeout is the maximum total time allowed for a task",
	JobDefaultsBatchTaskTimeoutsExecutionTimeoutKey:  "ExecutionTimeout is the maximum time allowed for task execution",
	JobDefaultsOpsPriorityKey:                        "Priority specifies the default priority allocated to a service or daemon job. This value is used when the job hasn't explicitly set its priority requirement.",
	JobDefaultsOpsTaskResourcesCPUKey:                "CPU specifies the default amount of CPU allocated to a task. It uses Kubernetes resource string format (e.g., \"100m\" for 0.1 CPU cores). This value is used when the task hasn't explicitly set its CPU requirement.",
	JobDefaultsOpsTaskResourcesMemoryKey:             "Memory specifies the default amount of memory allocated to a task. It uses Kubernetes resource string format (e.g., \"256Mi\" for 256 mebibytes). This value is used when the task hasn't explicitly set its memory requirement.",
	JobDefaultsOpsTaskResourcesDiskKey:               "Disk specifies the default amount of disk space allocated to a task. It uses Kubernetes resource string format (e.g., \"1Gi\" for 1 gibibyte). This value is used when the task hasn't explicitly set its disk space requirement.",
	JobDefaultsOpsTaskResourcesGPUKey:                "GPU specifies the default number of GPUs allocated to a task. It uses Kubernetes resource string format (e.g., \"1\" for 1 GPU). This value is used when the task hasn't explicitly set its GPU requirement.",
	JobDefaultsOpsTaskPublisherConfigTypeKey:         "No description available",
	JobDefaultsOpsTaskPublisherConfigParamsKey:       "No description available",
	JobDefaultsOpsTaskTimeoutsTotalTimeoutKey:        "TotalTimeout is the maximum total time allowed for a task",
	JobDefaultsOpsTaskTimeoutsExecutionTimeoutKey:    "ExecutionTimeout is the maximum time allowed for task execution",
	JobDefaultsDaemonPriorityKey:                     "Priority specifies the default priority allocated to a service or daemon job. This value is used when the job hasn't explicitly set its priority requirement.",
	JobDefaultsDaemonTaskResourcesCPUKey:             "CPU specifies the default amount of CPU allocated to a task. It uses Kubernetes resource string format (e.g., \"100m\" for 0.1 CPU cores). This value is used when the task hasn't explicitly set its CPU requirement.",
	JobDefaultsDaemonTaskResourcesMemoryKey:          "Memory specifies the default amount of memory allocated to a task. It uses Kubernetes resource string format (e.g., \"256Mi\" for 256 mebibytes). This value is used when the task hasn't explicitly set its memory requirement.",
	JobDefaultsDaemonTaskResourcesDiskKey:            "Disk specifies the default amount of disk space allocated to a task. It uses Kubernetes resource string format (e.g., \"1Gi\" for 1 gibibyte). This value is used when the task hasn't explicitly set its disk space requirement.",
	JobDefaultsDaemonTaskResourcesGPUKey:             "GPU specifies the default number of GPUs allocated to a task. It uses Kubernetes resource string format (e.g., \"1\" for 1 GPU). This value is used when the task hasn't explicitly set its GPU requirement.",
	JobDefaultsServicePriorityKey:                    "Priority specifies the default priority allocated to a service or daemon job. This value is used when the job hasn't explicitly set its priority requirement.",
	JobDefaultsServiceTaskResourcesCPUKey:            "CPU specifies the default amount of CPU allocated to a task. It uses Kubernetes resource string format (e.g., \"100m\" for 0.1 CPU cores). This value is used when the task hasn't explicitly set its CPU requirement.",
	JobDefaultsServiceTaskResourcesMemoryKey:         "Memory specifies the default amount of memory allocated to a task. It uses Kubernetes resource string format (e.g., \"256Mi\" for 256 mebibytes). This value is used when the task hasn't explicitly set its memory requirement.",
	JobDefaultsServiceTaskResourcesDiskKey:           "Disk specifies the default amount of disk space allocated to a task. It uses Kubernetes resource string format (e.g., \"1Gi\" for 1 gibibyte). This value is used when the task hasn't explicitly set its disk space requirement.",
	JobDefaultsServiceTaskResourcesGPUKey:            "GPU specifies the default number of GPUs allocated to a task. It uses Kubernetes resource string format (e.g., \"1\" for 1 GPU). This value is used when the task hasn't explicitly set its GPU requirement.",
	JobAdmissionControlRejectStatelessJobsKey:        "RejectStatelessJobs indicates whether to reject stateless jobs, i.e. jobs without inputs.",
	JobAdmissionControlAcceptNetworkedJobsKey:        "AcceptNetworkedJobs indicates whether to accept jobs that require network access.",
	JobAdmissionControlProbeHTTPKey:                  "ProbeHTTP specifies the HTTP endpoint for probing job submission.",
	JobAdmissionControlProbeExecKey:                  "ProbeExec specifies the command to execute for probing job submission.",
	LoggingLevelKey:                                  "Level sets the logging level. One of: trace, debug, info, warn, error, fatal, panic.",
	LoggingModeKey:                                   "Mode specifies the logging mode. One of: default, json.",
	LoggingLogDebugInfoIntervalKey:                   "LogDebugInfoInterval specifies the interval for logging debug information.",
	UpdateConfigIntervalKey:                          "Interval specifies the time between update checks, when set to 0 update checks are not performed.",
	FeatureFlagsExecTranslationKey:                   "ExecTranslation enables the execution translation feature.",
	DisableAnalyticsKey:                              "No description available",
}

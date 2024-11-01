// CODE GENERATED BY pkg/config/types/gen/generate.go DO NOT EDIT

package types

// ConfigDescriptions maps configuration paths to their descriptions
var ConfigDescriptions = map[string]string{
	APIAuthAccessPolicyPathKey:                       "AccessPolicyPath is the path to a file or directory that will be loaded as the policy to apply to all inbound API requests. If unspecified, a policy that permits access to all API endpoints to both authenticated and unauthenticated users (the default as of v1.2.0) will be used.",
	APIAuthMethodsKey:                                "Methods maps \"method names\" to authenticator implementations. A method name is a human-readable string chosen by the person configuring the system that is shown to users to help them pick the authentication method they want to use. There can be multiple usages of the same Authenticator *type* but with different configs and parameters, each identified with a unique method name.  For example, if an implementation wants to allow users to log in with Github or Bitbucket, they might both use an authenticator implementation of type \"oidc\", and each would appear once on this provider with key / method name \"github\" and \"bitbucket\".  By default, only a single authentication method that accepts authentication via client keys will be enabled.",
	APIHostKey:                                       "Host specifies the hostname or IP address on which the API server listens or the client connects.",
	APIPortKey:                                       "Port specifies the port number on which the API server listens or the client connects.",
	APITLSAutoCertKey:                                "AutoCert specifies the domain for automatic certificate generation.",
	APITLSAutoCertCachePathKey:                       "AutoCertCachePath specifies the directory to cache auto-generated certificates.",
	APITLSCAFileKey:                                  "CAFile specifies the path to the Certificate Authority file.",
	APITLSCertFileKey:                                "CertFile specifies the path to the TLS certificate file.",
	APITLSInsecureKey:                                "Insecure allows insecure TLS connections (e.g., self-signed certificates).",
	APITLSKeyFileKey:                                 "KeyFile specifies the path to the TLS private key file.",
	APITLSSelfSignedKey:                              "SelfSigned indicates whether to use a self-signed certificate.",
	APITLSUseTLSKey:                                  "UseTLS indicates whether to use TLS for client connections.",
	ComputeAllocatedCapacityCPUKey:                   "CPU specifies the amount of CPU a compute node allocates for running jobs. It can be expressed as a percentage (e.g., \"85%\") or a Kubernetes resource string (e.g., \"100m\").",
	ComputeAllocatedCapacityDiskKey:                  "Disk specifies the amount of Disk space a compute node allocates for running jobs. It can be expressed as a percentage (e.g., \"85%\") or a Kubernetes resource string (e.g., \"10Gi\").",
	ComputeAllocatedCapacityGPUKey:                   "GPU specifies the amount of GPU a compute node allocates for running jobs. It can be expressed as a percentage (e.g., \"85%\") or a Kubernetes resource string (e.g., \"1\"). Note: When using percentages, the result is always rounded up to the nearest whole GPU.",
	ComputeAllocatedCapacityMemoryKey:                "Memory specifies the amount of Memory a compute node allocates for running jobs. It can be expressed as a percentage (e.g., \"85%\") or a Kubernetes resource string (e.g., \"1Gi\").",
	ComputeAllowListedLocalPathsKey:                  "AllowListedLocalPaths specifies a list of local file system paths that the compute node is allowed to access.",
	ComputeAuthTokenKey:                              "Token specifies the key for compute nodes to be able to access the orchestrator.",
	ComputeEnabledKey:                                "Enabled indicates whether the compute node is active and available for job execution.",
	ComputeHeartbeatInfoUpdateIntervalKey:            "InfoUpdateInterval specifies the time between updates of non-resource information to the orchestrator.",
	ComputeHeartbeatIntervalKey:                      "Interval specifies the time between heartbeat signals sent to the orchestrator.",
	ComputeHeartbeatResourceUpdateIntervalKey:        "ResourceUpdateInterval specifies the time between updates of resource information to the orchestrator.",
	ComputeNATSCACertKey:                             "CACert specifies the CA file path that the compute node trusts when connecting to NATS server.",
	ComputeOrchestratorsKey:                          "Orchestrators specifies a list of orchestrator endpoints that this compute node connects to.",
	DataDirKey:                                       "DataDir specifies a location on disk where the bacalhau node will maintain state.",
	DisableAnalyticsKey:                              "DisableAnalytics, when true, disables sharing anonymous analytics data with the Bacalhau development team",
	EnginesDisabledKey:                               "Disabled specifies a list of engines that are disabled.",
	EnginesTypesDockerManifestCacheRefreshKey:        "Refresh specifies the refresh interval for cache entries.",
	EnginesTypesDockerManifestCacheSizeKey:           "Size specifies the size of the Docker manifest cache.",
	EnginesTypesDockerManifestCacheTTLKey:            "TTL specifies the time-to-live duration for cache entries.",
	InputSourcesDisabledKey:                          "Disabled specifies a list of storages that are disabled.",
	InputSourcesMaxRetryCountKey:                     "ReadTimeout specifies the maximum number of attempts for reading from a storage.",
	InputSourcesReadTimeoutKey:                       "ReadTimeout specifies the maximum time allowed for reading from a storage.",
	InputSourcesTypesIPFSEndpointKey:                 "Endpoint specifies the multi-address to connect to for IPFS. e.g /ip4/127.0.0.1/tcp/5001",
	JobAdmissionControlAcceptNetworkedJobsKey:        "AcceptNetworkedJobs indicates whether to accept jobs that require network access.",
	JobAdmissionControlLocalityKey:                   "Locality specifies the locality of the job input data.",
	JobAdmissionControlProbeExecKey:                  "ProbeExec specifies the command to execute for probing job submission.",
	JobAdmissionControlProbeHTTPKey:                  "ProbeHTTP specifies the HTTP endpoint for probing job submission.",
	JobAdmissionControlRejectStatelessJobsKey:        "RejectStatelessJobs indicates whether to reject stateless jobs, i.e. jobs without inputs.",
	JobDefaultsBatchPriorityKey:                      "Priority specifies the default priority allocated to a batch or ops job. This value is used when the job hasn't explicitly set its priority requirement.",
	JobDefaultsBatchTaskPublisherParamsKey:           "Params specifies the publisher configuration data.",
	JobDefaultsBatchTaskPublisherTypeKey:             "Type specifies the publisher type. e.g. \"s3\", \"local\", \"ipfs\", etc.",
	JobDefaultsBatchTaskResourcesCPUKey:              "CPU specifies the default amount of CPU allocated to a task. It uses Kubernetes resource string format (e.g., \"100m\" for 0.1 CPU cores). This value is used when the task hasn't explicitly set its CPU requirement.",
	JobDefaultsBatchTaskResourcesDiskKey:             "Disk specifies the default amount of disk space allocated to a task. It uses Kubernetes resource string format (e.g., \"1Gi\" for 1 gibibyte). This value is used when the task hasn't explicitly set its disk space requirement.",
	JobDefaultsBatchTaskResourcesGPUKey:              "GPU specifies the default number of GPUs allocated to a task. It uses Kubernetes resource string format (e.g., \"1\" for 1 GPU). This value is used when the task hasn't explicitly set its GPU requirement.",
	JobDefaultsBatchTaskResourcesMemoryKey:           "Memory specifies the default amount of memory allocated to a task. It uses Kubernetes resource string format (e.g., \"256Mi\" for 256 mebibytes). This value is used when the task hasn't explicitly set its memory requirement.",
	JobDefaultsBatchTaskTimeoutsExecutionTimeoutKey:  "ExecutionTimeout is the maximum time allowed for task execution",
	JobDefaultsBatchTaskTimeoutsTotalTimeoutKey:      "TotalTimeout is the maximum total time allowed for a task",
	JobDefaultsDaemonPriorityKey:                     "Priority specifies the default priority allocated to a service or daemon job. This value is used when the job hasn't explicitly set its priority requirement.",
	JobDefaultsDaemonTaskResourcesCPUKey:             "CPU specifies the default amount of CPU allocated to a task. It uses Kubernetes resource string format (e.g., \"100m\" for 0.1 CPU cores). This value is used when the task hasn't explicitly set its CPU requirement.",
	JobDefaultsDaemonTaskResourcesDiskKey:            "Disk specifies the default amount of disk space allocated to a task. It uses Kubernetes resource string format (e.g., \"1Gi\" for 1 gibibyte). This value is used when the task hasn't explicitly set its disk space requirement.",
	JobDefaultsDaemonTaskResourcesGPUKey:             "GPU specifies the default number of GPUs allocated to a task. It uses Kubernetes resource string format (e.g., \"1\" for 1 GPU). This value is used when the task hasn't explicitly set its GPU requirement.",
	JobDefaultsDaemonTaskResourcesMemoryKey:          "Memory specifies the default amount of memory allocated to a task. It uses Kubernetes resource string format (e.g., \"256Mi\" for 256 mebibytes). This value is used when the task hasn't explicitly set its memory requirement.",
	JobDefaultsOpsPriorityKey:                        "Priority specifies the default priority allocated to a batch or ops job. This value is used when the job hasn't explicitly set its priority requirement.",
	JobDefaultsOpsTaskPublisherParamsKey:             "Params specifies the publisher configuration data.",
	JobDefaultsOpsTaskPublisherTypeKey:               "Type specifies the publisher type. e.g. \"s3\", \"local\", \"ipfs\", etc.",
	JobDefaultsOpsTaskResourcesCPUKey:                "CPU specifies the default amount of CPU allocated to a task. It uses Kubernetes resource string format (e.g., \"100m\" for 0.1 CPU cores). This value is used when the task hasn't explicitly set its CPU requirement.",
	JobDefaultsOpsTaskResourcesDiskKey:               "Disk specifies the default amount of disk space allocated to a task. It uses Kubernetes resource string format (e.g., \"1Gi\" for 1 gibibyte). This value is used when the task hasn't explicitly set its disk space requirement.",
	JobDefaultsOpsTaskResourcesGPUKey:                "GPU specifies the default number of GPUs allocated to a task. It uses Kubernetes resource string format (e.g., \"1\" for 1 GPU). This value is used when the task hasn't explicitly set its GPU requirement.",
	JobDefaultsOpsTaskResourcesMemoryKey:             "Memory specifies the default amount of memory allocated to a task. It uses Kubernetes resource string format (e.g., \"256Mi\" for 256 mebibytes). This value is used when the task hasn't explicitly set its memory requirement.",
	JobDefaultsOpsTaskTimeoutsExecutionTimeoutKey:    "ExecutionTimeout is the maximum time allowed for task execution",
	JobDefaultsOpsTaskTimeoutsTotalTimeoutKey:        "TotalTimeout is the maximum total time allowed for a task",
	JobDefaultsServicePriorityKey:                    "Priority specifies the default priority allocated to a service or daemon job. This value is used when the job hasn't explicitly set its priority requirement.",
	JobDefaultsServiceTaskResourcesCPUKey:            "CPU specifies the default amount of CPU allocated to a task. It uses Kubernetes resource string format (e.g., \"100m\" for 0.1 CPU cores). This value is used when the task hasn't explicitly set its CPU requirement.",
	JobDefaultsServiceTaskResourcesDiskKey:           "Disk specifies the default amount of disk space allocated to a task. It uses Kubernetes resource string format (e.g., \"1Gi\" for 1 gibibyte). This value is used when the task hasn't explicitly set its disk space requirement.",
	JobDefaultsServiceTaskResourcesGPUKey:            "GPU specifies the default number of GPUs allocated to a task. It uses Kubernetes resource string format (e.g., \"1\" for 1 GPU). This value is used when the task hasn't explicitly set its GPU requirement.",
	JobDefaultsServiceTaskResourcesMemoryKey:         "Memory specifies the default amount of memory allocated to a task. It uses Kubernetes resource string format (e.g., \"256Mi\" for 256 mebibytes). This value is used when the task hasn't explicitly set its memory requirement.",
	LabelsKey:                                        "Labels are key-value pairs used to describe and categorize the nodes.",
	LoggingLevelKey:                                  "Level sets the logging level. One of: trace, debug, info, warn, error, fatal, panic.",
	LoggingLogDebugInfoIntervalKey:                   "LogDebugInfoInterval specifies the interval for logging debug information.",
	LoggingModeKey:                                   "Mode specifies the logging mode. One of: default, json.",
	NameProviderKey:                                  "NameProvider specifies the method used to generate names for the node. One of: hostname, aws, gcp, uuid, puuid.",
	OrchestratorAdvertiseKey:                         "Advertise specifies URL to advertise to other servers.",
	OrchestratorAuthTokenKey:                         "Token specifies the key for compute nodes to be able to access the orchestrator",
	OrchestratorClusterAdvertiseKey:                  "Advertise specifies the address to advertise to other cluster members.",
	OrchestratorClusterHostKey:                       "Host specifies the hostname or IP address for cluster communication.",
	OrchestratorClusterNameKey:                       "Name specifies the unique identifier for this orchestrator cluster.",
	OrchestratorClusterPeersKey:                      "Peers is a list of other cluster members to connect to on startup.",
	OrchestratorClusterPortKey:                       "Port specifies the port number for cluster communication.",
	OrchestratorEnabledKey:                           "Enabled indicates whether the orchestrator node is active and available for job submission.",
	OrchestratorEvaluationBrokerMaxRetryCountKey:     "MaxRetryCount specifies the maximum number of times an evaluation can be retried before being marked as failed.",
	OrchestratorEvaluationBrokerVisibilityTimeoutKey: "VisibilityTimeout specifies how long an evaluation can be claimed before it's returned to the queue.",
	OrchestratorHostKey:                              "Host specifies the hostname or IP address on which the Orchestrator server listens for compute node connections.",
	OrchestratorNATSCACertKey:                        "CACert specifies the CA file path that the orchestrator node trusts when connecting to NATS server.",
	OrchestratorNATSServerTLSCertKey:                 "ServerTLSCert specifies the certificate file path given to NATS server to serve TLS connections.",
	OrchestratorNATSServerTLSKeyKey:                  "ServerTLSKey specifies the private key file path given to NATS server to serve TLS connections.",
	OrchestratorNATSServerTLSTimeoutKey:              "ServerTLSTimeout specifies the TLS timeout, in seconds, set on the NATS server.",
	OrchestratorNodeManagerDisconnectTimeoutKey:      "DisconnectTimeout specifies how long to wait before considering a node disconnected.",
	OrchestratorNodeManagerManualApprovalKey:         "ManualApproval, if true, requires manual approval for new compute nodes joining the cluster.",
	OrchestratorPortKey:                              "Host specifies the port number on which the Orchestrator server listens for compute node connections.",
	OrchestratorSchedulerHousekeepingIntervalKey:     "HousekeepingInterval specifies how often to run housekeeping tasks.",
	OrchestratorSchedulerHousekeepingTimeoutKey:      "HousekeepingTimeout specifies the maximum time allowed for a single housekeeping run.",
	OrchestratorSchedulerQueueBackoffKey:             "QueueBackoff specifies the time to wait before retrying a failed job.",
	OrchestratorSchedulerWorkerCountKey:              "WorkerCount specifies the number of concurrent workers for job scheduling.",
	PublishersDisabledKey:                            "Disabled specifies a list of publishers that are disabled.",
	PublishersTypesIPFSEndpointKey:                   "Endpoint specifies the multi-address to connect to for IPFS. e.g /ip4/127.0.0.1/tcp/5001",
	PublishersTypesLocalAddressKey:                   "Address specifies the endpoint the publisher serves on.",
	PublishersTypesLocalPortKey:                      "Port specifies the port the publisher serves on.",
	PublishersTypesS3PreSignedURLDisabledKey:         "PreSignedURLDisabled specifies whether pre-signed URLs are enabled for the S3 provider.",
	PublishersTypesS3PreSignedURLExpirationKey:       "PreSignedURLExpiration specifies the duration before a pre-signed URL expires.",
	ResultDownloadersDisabledKey:                     "Disabled is a list of downloaders that are disabled.",
	ResultDownloadersTimeoutKey:                      "Timeout specifies the maximum time allowed for a download operation.",
	ResultDownloadersTypesIPFSEndpointKey:            "Endpoint specifies the multi-address to connect to for IPFS. e.g /ip4/127.0.0.1/tcp/5001",
	StrictVersionMatchKey:                            "StrictVersionMatch indicates whether to enforce strict version matching.",
	UpdateConfigIntervalKey:                          "Interval specifies the time between update checks, when set to 0 update checks are not performed.",
	WebUIBackendKey:                                  "Backend specifies the address and port of the backend API server. If empty, the Web UI will use the same address and port as the API server.",
	WebUIEnabledKey:                                  "Enabled indicates whether the Web UI is enabled.",
	WebUIListenKey:                                   "Listen specifies the address and port on which the Web UI listens.",
}

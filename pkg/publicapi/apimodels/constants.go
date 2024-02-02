package apimodels

const (
	// AllNamespacesNamespace is a sentinel Namespace value to indicate that api should search for
	// jobs and allocations in all the namespaces the requester can access.
	AllNamespacesNamespace = "*"

	// HTTPHeaderClientID is the header used to pass the client ID to the server.
	HTTPHeaderClientID = "X-Bacalhau-Client-ID"

	// HTTPHeaderJobID is the header used to pass the job ID to the server.
	HTTPHeaderJobID = "X-Bacalhau-Job-ID"

	// HTTPHeaderAppID is the header used to pass the application ID to the server.
	HTTPHeaderAppID = "X-Bacalhau-App-ID"

	// HTTPHeaderBacalhauGitVersion is the header used to pass the agent version, eg v1.2.3
	HTTPHeaderBacalhauGitVersion = "X-Bacalhau-Git-Version"
	// HTTPHeaderBacalhauGitCommit is the header used to pass the agent git commit
	HTTPHeaderBacalhauGitCommit = "X-Bacalhau-Git-Commit"
	// HTTPHeaderBacalhauBuildDate is the header used to pass the agent build date in UTC
	HTTPHeaderBacalhauBuildDate = "X-Bacalhau-Build-Date"
	// HTTPHeaderBacalhauBuildOS is the header used to pass the agent operating system
	HTTPHeaderBacalhauBuildOS = "X-Bacalhau-Build-OS"
	// HTTPHeaderBacalhauArch is the header used to pass the agent architecture
	HTTPHeaderBacalhauArch = "X-Bacalhau-Arch"
)

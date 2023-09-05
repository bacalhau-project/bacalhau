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
)

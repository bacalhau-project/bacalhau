package analytics

// Node identification property keys used in telemetry events
const (
	// NodeInstallationIDKey is the key for the installation ID property
	NodeInstallationIDKey = "installation_id"

	// NodeInstanceIDKey is the key for the instance ID property
	NodeInstanceIDKey = "instance_id"

	// NodeIDHashKey is the key for the hashed node ID property
	NodeIDHashKey = "node_id_hash"

	// NodeTypeKey is the key for the node type property (hybrid, orchestrator, compute)
	NodeTypeKey = "node_type"

	// NodeVersionKey is the key for the node software version property
	NodeVersionKey = "node_version"

	// NetworkIDKey is the key for the network ID property
	NetworkIDKey = "network_id"

	// AccountIDKey is the key for the account ID property (in Expanso Cloud)
	AccountIDKey = "account_id"
)

// Environment property keys used in telemetry events
const (
	// EnvKey is the key for the environment type property
	EnvKey = "environment"

	// EnvOSTypeKey is the key for the operating system type property
	EnvOSTypeKey = "os_type"

	// EnvOSArchKey is the key for the operating system architecture property
	EnvOSArchKey = "os_arch"
)

// Environment type values used in telemetry events
const (
	// EnvDockerVal indicates the node is running in a Docker container
	EnvDockerVal = "docker"

	// EnvExpansoCloudVal indicates the node is running in Expanso Cloud
	EnvExpansoCloudVal = "expanso"

	// EnvLocalVal indicates the node is running in a local environment
	EnvLocalVal = "local"
)

// ResourceAttributesKey is the container key for all resource attributes
// in telemetry event properties
const ResourceAttributesKey = "resource_attributes"

// ResourceAttributes contains all the node-level properties that should be
// included with every telemetry event. These attributes provide context
// about the node's identity, environment, and configuration.
type ResourceAttributes struct {
	// Node identification
	NodeVersion    string `json:"node_version,omitempty"`    // Version of the node software
	InstallationID string `json:"installation_id,omitempty"` // Persistent ID for this installation
	InstanceID     string `json:"instance_id,omitempty"`     // ID for this specific instance of the node
	NodeIDHash     string `json:"node_id_hash,omitempty"`    // Hashed node ID for anonymity
	NodeType       string `json:"node_type,omitempty"`       // Role of the node (hybrid, orchestrator, compute)
	NetworkID      string `json:"network_id,omitempty"`      // ID of the network this node belongs to
	AccountID      string `json:"account_id,omitempty"`      // ID of the account (for Expanso Cloud)

	// Environment information
	Environment string `json:"environment,omitempty"` // Environment type (docker, expanso, local)
	OSType      string `json:"os_type,omitempty"`     // Operating system type (linux, darwin, windows)
	OSArch      string `json:"os_arch,omitempty"`     // CPU architecture (amd64, arm64, etc.)
}

// DetermineDistinctID returns the distinct ID for telemetry events
// based on the node properties. It uses a fallback hierarchy:
// 1. Account ID (highest priority)
// 2. Network ID
// 3. Installation ID
// 4. Instance ID
// 5. "unknown" (if no IDs are available)
func (attrs *ResourceAttributes) DetermineDistinctID() string {
	if attrs.AccountID != "" {
		return attrs.AccountID
	}
	if attrs.NetworkID != "" {
		return attrs.NetworkID
	}
	if attrs.InstallationID != "" {
		return attrs.InstallationID
	}
	if attrs.InstanceID != "" {
		return attrs.InstanceID
	}
	return "unknown"
}

// ApplyDefaults ensures that all resource attributes have appropriate values
// by applying defaults and fallbacks where necessary.
// Currently, it sets NetworkID to InstanceID if NetworkID is not set.
func (attrs *ResourceAttributes) ApplyDefaults() {
	if attrs.NetworkID == "" && attrs.InstanceID != "" {
		attrs.NetworkID = attrs.InstanceID
	}
}

// Properties converts the ResourceAttributes to a map structure suitable
// for telemetry events. The attributes are nested under the ResourceAttributesKey
// to keep them organized within the event properties.
//
// The returned structure is:
//
//	{
//	    "resource_attributes": {
//	        "installation_id": "...",
//	        "instance_id": "...",
//	        ...
//	    }
//	}
func (attrs *ResourceAttributes) Properties() EventProperties {
	props := make(map[string]interface{})

	// Only add non-empty attributes to the properties map
	if attrs.InstallationID != "" {
		props[NodeInstallationIDKey] = attrs.InstallationID
	}
	if attrs.InstanceID != "" {
		props[NodeInstanceIDKey] = attrs.InstanceID
	}
	if attrs.NodeIDHash != "" {
		props[NodeIDHashKey] = attrs.NodeIDHash
	}
	if attrs.NodeType != "" {
		props[NodeTypeKey] = attrs.NodeType
	}
	if attrs.NodeVersion != "" {
		props[NodeVersionKey] = attrs.NodeVersion
	}
	if attrs.NetworkID != "" {
		props[NetworkIDKey] = attrs.NetworkID
	}
	if attrs.AccountID != "" {
		props[AccountIDKey] = attrs.AccountID
	}
	if attrs.Environment != "" {
		props[EnvKey] = attrs.Environment
	}
	if attrs.OSType != "" {
		props[EnvOSTypeKey] = attrs.OSType
	}
	if attrs.OSArch != "" {
		props[EnvOSArchKey] = attrs.OSArch
	}

	// Nest all properties under ResourceAttributesKey
	return EventProperties{
		ResourceAttributesKey: props,
	}
}

package analytics

// Node identification property keys used in analytics events
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
)

// Environment property keys used in analytics events
const (
	// EnvKey is the key for the environment type property
	EnvKey = "environment"

	// EnvOSTypeKey is the key for the operating system type property
	EnvOSTypeKey = "os_type"

	// EnvOSArchKey is the key for the operating system architecture property
	EnvOSArchKey = "os_arch"
)

// Environment type values used in analytics events
const (
	// EnvDockerVal indicates the node is running in a Docker container
	EnvDockerVal = "docker"

	// EnvLocalVal indicates the node is running in a local environment
	EnvLocalVal = "local"
)

// ResourceAttributesKey is the container key for all resource attributes
// in analytics event properties
const ResourceAttributesKey = "resource_attributes"

// ResourceAttributes contains all the node-level properties that should be
// included with every analytics event. These attributes provide context
// about the node's identity, environment, and configuration.
type ResourceAttributes struct {
	// Node identification
	NodeVersion    string `json:"node_version,omitempty"`    // Version of the node software
	InstallationID string `json:"installation_id,omitempty"` // Persistent ID for this installation
	InstanceID     string `json:"instance_id,omitempty"`     // ID for this specific instance of the node
	NodeIDHash     string `json:"node_id_hash,omitempty"`    // Hashed node ID for anonymity
	NodeType       string `json:"node_type,omitempty"`       // Role of the node (hybrid, orchestrator, compute)
	NetworkID      string `json:"network_id,omitempty"`      // ID of the network this node belongs to

	// Environment information
	Environment string `json:"environment,omitempty"` // Environment type (docker, local)
	OSType      string `json:"os_type,omitempty"`     // Operating system type (linux, darwin, windows)
	OSArch      string `json:"os_arch,omitempty"`     // CPU architecture (amd64, arm64, etc.)
}

// DetermineDistinctID returns the distinct ID for analytics events
// based on the node properties. It uses a fallback hierarchy:
// 1. Network ID (highest priority)
// 2. Installation ID
// 3. Instance ID
// 4. "unknown" (if no IDs are available)
func (attrs *ResourceAttributes) DetermineDistinctID() string {
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
// for analytics events. The attributes are nested under the ResourceAttributesKey
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

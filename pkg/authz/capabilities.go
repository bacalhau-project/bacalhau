package authz

import (
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

// CapabilityChecker is responsible for checking if a user has the required capabilities
// for a specific resource type and HTTP method.
type CapabilityChecker struct{}

// NewCapabilityChecker creates a new instance of CapabilityChecker
func NewCapabilityChecker() *CapabilityChecker {
	return &CapabilityChecker{}
}

// ResourceType represents the type of resource being accessed
type ResourceType string

const (
	ResourceTypeNode  ResourceType = "node"
	ResourceTypeJob   ResourceType = "job"
	ResourceTypeAgent ResourceType = "agent"
	ResourceTypeOpen  ResourceType = "open"
)

// GetRequiredCapability determines the required capability for a specific resource type and HTTP method
func (c *CapabilityChecker) GetRequiredCapability(resourceType ResourceType, method string) string {
	isReadOperation := method == http.MethodGet

	switch resourceType {
	case ResourceTypeNode:
		if isReadOperation {
			return "read:node"
		}
		return "write:node"
	case ResourceTypeJob:
		if isReadOperation {
			return "read:job"
		}
		return "write:job"
	case ResourceTypeAgent:
		if isReadOperation {
			return "read:agent"
		}
		return "write:agent"
	default:
		// If no resource type matched, default to requiring node admin for safety
		return "write:node"
	}
}

// HasRequiredCapability checks if a user has the required capability
func (c *CapabilityChecker) HasRequiredCapability(user types.AuthUser, requiredCapability string) bool {
	// Guard against empty capabilities
	if requiredCapability == "" {
		return false
	}

	// Determine if it's a read operation based on the required capability
	// Only consider it a read operation if it explicitly starts with "read:"
	isReadOperation := len(requiredCapability) >= 5 && requiredCapability[:5] == "read:"

	// Check against the exact list of allowed capabilities
	for _, capability := range user.Capabilities {
		for _, action := range capability.Actions {
			// Universal wildcard
			if action == "*" {
				return true
			}

			// Resource-specific capability
			if action == requiredCapability {
				return true
			}

			// Read wildcard
			if isReadOperation && action == "read:*" {
				return true
			}

			// Write wildcard - matches any non-read: capability
			if !isReadOperation && action == "write:*" {
				return true
			}
		}
	}

	return false
}

// CheckUserAccess checks if a user has access to a resource for a given HTTP method
// Returns whether the user has access, the required capability, and any error
func (c *CapabilityChecker) CheckUserAccess(user types.AuthUser, resourceType ResourceType, req *http.Request) (bool, string, error) {
	// Get the required capability
	requiredCapability := c.GetRequiredCapability(resourceType, req.Method)

	// Check if user has the required capability
	hasCapability := c.HasRequiredCapability(user, requiredCapability)

	return hasCapability, requiredCapability, nil
}

// MapEndpointToResourceType maps an API endpoint path to a resource type
func MapEndpointToResourceType(path string, endpointsPermissions map[string]string) ResourceType {
	var longestMatch string
	var matchedResourceType string

	// Find the longest matching endpoint pattern
	for endpoint, permission := range endpointsPermissions {
		if endpoint != "" && path != "" &&
			len(endpoint) <= len(path) &&
			path[:len(endpoint)] == endpoint &&
			len(endpoint) > len(longestMatch) {
			longestMatch = endpoint
			matchedResourceType = permission
		}
	}

	// Return the resource type for the longest match (or empty if no match)
	return ResourceType(matchedResourceType)
}

// GetDefaultEndpointPermissions returns the default endpoint to permission mapping
func GetDefaultEndpointPermissions() map[string]string {
	return map[string]string{
		"/api/v1/auth":    "open",
		"/api/v1/version": "open",

		"/api/v1/agent":            "agent",
		"/api/v1/agent/alive":      "open",
		"/api/v1/agent/version":    "open",
		"/api/v1/agent/authconfig": "open",

		"/api/v1/orchestrator/jobs":  "job",
		"/api/v1/orchestrator/nodes": "node",
	}
}

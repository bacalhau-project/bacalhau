package authz

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/stretchr/testify/assert"
)

// TestGetRequiredCapability verifies that the correct capability is returned for different resource types and HTTP methods
func TestGetRequiredCapability(t *testing.T) {
	t.Run("Node Read", func(t *testing.T) {
		checker := NewCapabilityChecker()
		capability := checker.GetRequiredCapability(ResourceTypeNode, http.MethodGet)
		assert.Equal(t, "read:node", capability)
	})

	t.Run("Node Write", func(t *testing.T) {
		checker := NewCapabilityChecker()
		capability := checker.GetRequiredCapability(ResourceTypeNode, http.MethodPost)
		assert.Equal(t, "write:node", capability)
	})

	t.Run("Job Read", func(t *testing.T) {
		checker := NewCapabilityChecker()
		capability := checker.GetRequiredCapability(ResourceTypeJob, http.MethodGet)
		assert.Equal(t, "read:job", capability)
	})

	t.Run("Job Write", func(t *testing.T) {
		checker := NewCapabilityChecker()
		capability := checker.GetRequiredCapability(ResourceTypeJob, http.MethodPost)
		assert.Equal(t, "write:job", capability)
	})

	t.Run("Unknown Resource Type", func(t *testing.T) {
		checker := NewCapabilityChecker()
		capability := checker.GetRequiredCapability("unknown", http.MethodGet)
		assert.Equal(t, "write:node", capability, "Unknown resource type should default to write:node")
	})
}

// TestHasRequiredCapability verifies that capability checking works for various capability patterns
func TestHasRequiredCapability(t *testing.T) {
	t.Run("Exact Match", func(t *testing.T) {
		checker := NewCapabilityChecker()
		user := types.AuthUser{
			Alias: "testuser",
			Capabilities: []types.Capability{
				{Actions: []string{"read:node"}},
			},
		}
		hasCapability := checker.HasRequiredCapability(user, "read:node")
		assert.True(t, hasCapability, "User should have exact matching capability")
	})

	t.Run("Universal Wildcard", func(t *testing.T) {
		checker := NewCapabilityChecker()
		user := types.AuthUser{
			Alias: "wildcard_user",
			Capabilities: []types.Capability{
				{Actions: []string{"*"}},
			},
		}
		hasCapability := checker.HasRequiredCapability(user, "write:job")
		assert.True(t, hasCapability, "User with * should have all capabilities")
	})

	t.Run("Read Wildcard", func(t *testing.T) {
		checker := NewCapabilityChecker()
		user := types.AuthUser{
			Alias: "read_user",
			Capabilities: []types.Capability{
				{Actions: []string{"read:*"}},
			},
		}

		// Should have read:node
		hasReadNode := checker.HasRequiredCapability(user, "read:node")
		assert.True(t, hasReadNode, "User with read:* should have read:node")

		// Should not have write:node
		hasWriteNode := checker.HasRequiredCapability(user, "write:node")
		assert.False(t, hasWriteNode, "User with read:* should not have write:node")
	})

	t.Run("Write Wildcard", func(t *testing.T) {
		checker := NewCapabilityChecker()
		user := types.AuthUser{
			Alias: "write_user",
			Capabilities: []types.Capability{
				{Actions: []string{"write:*"}},
			},
		}

		// Should have write:job
		hasWriteJob := checker.HasRequiredCapability(user, "write:job")
		assert.True(t, hasWriteJob, "User with write:* should have write:job")

		// Should not have read:job
		hasReadJob := checker.HasRequiredCapability(user, "read:job")
		assert.False(t, hasReadJob, "User with write:* should not have read:job")
	})

	t.Run("No Match", func(t *testing.T) {
		checker := NewCapabilityChecker()
		user := types.AuthUser{
			Alias: "limited_user",
			Capabilities: []types.Capability{
				{Actions: []string{"read:node"}},
			},
		}
		hasCapability := checker.HasRequiredCapability(user, "write:job")
		assert.False(t, hasCapability, "User should not have ungranted capability")
	})

	t.Run("Multiple Capabilities", func(t *testing.T) {
		checker := NewCapabilityChecker()
		user := types.AuthUser{
			Alias: "mixed_user",
			Capabilities: []types.Capability{
				{Actions: []string{"read:node", "write:node"}},
				{Actions: []string{"read:job"}},
			},
		}

		// Check various permissions
		assert.True(t, checker.HasRequiredCapability(user, "read:node"), "Should have read:node")
		assert.True(t, checker.HasRequiredCapability(user, "write:node"), "Should have write:node")
		assert.True(t, checker.HasRequiredCapability(user, "read:job"), "Should have read:job")
		assert.False(t, checker.HasRequiredCapability(user, "write:job"), "Should not have write:job")
	})
}

// TestCheckUserAccess verifies that the complete access checking flow works correctly
func TestCheckUserAccess(t *testing.T) {
	t.Run("Successful Access", func(t *testing.T) {
		checker := NewCapabilityChecker()
		user := types.AuthUser{
			Alias: "authorized_user",
			Capabilities: []types.Capability{
				{Actions: []string{"read:node", "write:node"}},
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/node_info", nil)
		hasAccess, requiredCapability := checker.CheckUserAccess(user, ResourceTypeNode, req)

		assert.True(t, hasAccess, "User should have access")
		assert.Equal(t, "read:node", requiredCapability)
	})

	t.Run("Denied Access", func(t *testing.T) {
		checker := NewCapabilityChecker()
		user := types.AuthUser{
			Alias: "unauthorized_user",
			Capabilities: []types.Capability{
				{Actions: []string{"read:node"}},
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/api/v1/node_info", nil)
		hasAccess, requiredCapability := checker.CheckUserAccess(user, ResourceTypeNode, req)

		assert.False(t, hasAccess, "User should not have access")
		assert.Equal(t, "write:node", requiredCapability)
	})
}

// TestMapEndpointToResourceType verifies that endpoint paths are correctly mapped to resource types
func TestMapEndpointToResourceType(t *testing.T) {
	// Create a test endpoint permissions map with both general and specific paths
	endpointPermissions := map[string]string{
		"/api/v1/agent":                 "agent",
		"/api/v1/agent/node":            "node",
		"/api/v1/agent/alive":           "open",
		"/api/v1/agent/debug":           "node",
		"/api/v1/jobs":                  "job",
		"/api/v1/orchestrator":          "node",
		"/api/v1/orchestrator/jobs":     "job",
		"/api/v1/orchestrator/jobs/123": "open",
	}

	t.Run("Exact Match", func(t *testing.T) {
		resourceType := MapEndpointToResourceType("/api/v1/agent/node", endpointPermissions)
		assert.Equal(t, ResourceTypeNode, resourceType)
	})

	t.Run("Prefix Match", func(t *testing.T) {
		resourceType := MapEndpointToResourceType("/api/v1/agent/node/details", endpointPermissions)
		assert.Equal(t, ResourceTypeNode, resourceType)
	})

	t.Run("Open Endpoint", func(t *testing.T) {
		resourceType := MapEndpointToResourceType("/api/v1/agent/alive", endpointPermissions)
		assert.Equal(t, ResourceTypeOpen, resourceType)
	})

	t.Run("No Match", func(t *testing.T) {
		resourceType := MapEndpointToResourceType("/api/v1/unknown", endpointPermissions)
		assert.Equal(t, ResourceType(""), resourceType)
	})

	t.Run("Empty Path", func(t *testing.T) {
		resourceType := MapEndpointToResourceType("", endpointPermissions)
		assert.Equal(t, ResourceType(""), resourceType)
	})

	// New tests for specific vs general path matching
	t.Run("Specific Path Takes Precedence", func(t *testing.T) {
		// Should match /api/v1/agent/alive (open) not /api/v1/agent (agent)
		resourceType := MapEndpointToResourceType("/api/v1/agent/alive", endpointPermissions)
		assert.Equal(t, ResourceTypeOpen, resourceType)
		assert.NotEqual(t, ResourceTypeAgent, resourceType)
	})

	t.Run("General Path When No Specific Match", func(t *testing.T) {
		// Should match /api/v1/agent (agent) as there's no more specific match
		resourceType := MapEndpointToResourceType("/api/v1/agent/unknown", endpointPermissions)
		assert.Equal(t, ResourceTypeAgent, resourceType)
	})

	t.Run("Path With Query Parameters", func(t *testing.T) {
		// Should still match the base path ignoring query params
		resourceType := MapEndpointToResourceType("/api/v1/agent/alive?param=value", endpointPermissions)
		assert.Equal(t, ResourceTypeOpen, resourceType)
	})

	t.Run("Nested Specific Paths", func(t *testing.T) {
		// Tests the most specific match in a three-level nesting
		generalType := MapEndpointToResourceType("/api/v1/orchestrator/unknown", endpointPermissions)
		assert.Equal(t, ResourceTypeNode, generalType, "Should match general orchestrator path")

		jobsType := MapEndpointToResourceType("/api/v1/orchestrator/jobs/456", endpointPermissions)
		assert.Equal(t, ResourceTypeJob, jobsType, "Should match jobs path")

		specificJobType := MapEndpointToResourceType("/api/v1/orchestrator/jobs/123/details", endpointPermissions)
		assert.Equal(t, ResourceTypeOpen, specificJobType, "Should match specific job ID path")
	})
}

// TestGetDefaultEndpointPermissions verifies that the default endpoint permissions are returned correctly
func TestGetDefaultEndpointPermissions(t *testing.T) {
	permissions := GetDefaultEndpointPermissions()

	// Test for a few key endpoints
	assert.Equal(t, "open", permissions["/api/v1/version"])
	assert.Equal(t, "agent", permissions["/api/v1/agent"])
	assert.Equal(t, "open", permissions["/api/v1/agent/alive"])
	assert.Equal(t, "node", permissions["/api/v1/orchestrator/nodes"])
	assert.Equal(t, "job", permissions["/api/v1/orchestrator/jobs"])

	// Ensure all important endpoints are covered
	assert.Greater(t, len(permissions), 7, "Default permissions should include all important endpoints")
}

// TestDefaultEndpointSpecificMatching tests that URL matching works correctly with the actual
// default endpoint permissions, prioritizing more specific paths over general ones
func TestDefaultEndpointSpecificMatching(t *testing.T) {
	// Get actual default permissions
	defaultPermissions := GetDefaultEndpointPermissions()

	// Test cases for specific paths that should override general paths
	testCases := []struct {
		name           string
		path           string
		expectedType   ResourceType
		unexpectedType ResourceType
	}{
		{
			name:           "Agent Alive Path",
			path:           "/api/v1/agent/alive",
			expectedType:   ResourceTypeOpen,
			unexpectedType: ResourceTypeAgent,
		},
		{
			name:           "Agent Version Path",
			path:           "/api/v1/agent/version",
			expectedType:   ResourceTypeOpen,
			unexpectedType: ResourceTypeAgent,
		},
		{
			name:           "Agent with Unknown Subpath",
			path:           "/api/v1/agent/unknown",
			expectedType:   ResourceTypeAgent,
			unexpectedType: ResourceTypeOpen,
		},
		{
			name:           "Agent Alive with Query Params",
			path:           "/api/v1/agent/alive?health=true&timeout=30",
			expectedType:   ResourceTypeOpen,
			unexpectedType: ResourceTypeAgent,
		},
		{
			name:           "Agent Alive with Extra Path Segments",
			path:           "/api/v1/agent/alive/details",
			expectedType:   ResourceTypeOpen,
			unexpectedType: ResourceTypeAgent,
		},
	}

	// Run all test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resourceType := MapEndpointToResourceType(tc.path, defaultPermissions)
			assert.Equal(t, tc.expectedType, resourceType,
				"Should map to the expected resource type")
			assert.NotEqual(t, tc.unexpectedType, resourceType,
				"Should not map to the unexpected resource type")
		})
	}
}

// TestResourceTypeAgent tests the capability checker with the ResourceTypeAgent resource type
func TestResourceTypeAgent(t *testing.T) {
	checker := NewCapabilityChecker()

	// Test read operation
	readCapability := checker.GetRequiredCapability(ResourceTypeAgent, http.MethodGet)
	assert.Equal(t, "read:agent", readCapability)

	// Test write operation
	writeCapability := checker.GetRequiredCapability(ResourceTypeAgent, http.MethodPost)
	assert.Equal(t, "write:agent", writeCapability)

	// Test user with agent capabilities
	user := types.AuthUser{
		Alias: "agent_user",
		Capabilities: []types.Capability{
			{Actions: []string{"read:agent", "write:agent"}},
		},
	}

	// Test read access
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent", nil)
	hasAccess, requiredCapability := checker.CheckUserAccess(user, ResourceTypeAgent, req)
	assert.True(t, hasAccess)
	assert.Equal(t, "read:agent", requiredCapability)

	// Test write access
	writeReq := httptest.NewRequest(http.MethodPost, "/api/v1/agent", nil)
	hasWriteAccess, requiredWriteCapability := checker.CheckUserAccess(user, ResourceTypeAgent, writeReq)
	assert.True(t, hasWriteAccess)
	assert.Equal(t, "write:agent", requiredWriteCapability)
}

// TestEdgeCaseEmptyCapabilities tests the capability checker with empty capabilities
func TestEdgeCaseEmptyCapabilities(t *testing.T) {
	checker := NewCapabilityChecker()

	// User with empty capabilities array
	userWithEmptyArray := types.AuthUser{
		Alias:        "empty_array_user",
		Capabilities: []types.Capability{},
	}

	// User with capability containing empty actions array
	userWithEmptyActions := types.AuthUser{
		Alias: "empty_actions_user",
		Capabilities: []types.Capability{
			{Actions: []string{}},
		},
	}

	// Tests for user with empty capabilities array
	assert.False(t, checker.HasRequiredCapability(userWithEmptyArray, "read:node"))

	// Tests for user with empty actions
	assert.False(t, checker.HasRequiredCapability(userWithEmptyActions, "read:node"))
}

// TestResourceTypeOpenDetection tests detection and handling of open resource types
func TestResourceTypeOpenDetection(t *testing.T) {
	defaultPermissions := GetDefaultEndpointPermissions()

	// Test endpoints that should map to open resource type
	openEndpoints := []string{
		"/api/v1/auth",
		"/api/v1/version",
		"/api/v1/agent/alive",
		"/api/v1/agent/version",
		"/api/v1/agent/authconfig",
	}

	for _, endpoint := range openEndpoints {
		t.Run(endpoint, func(t *testing.T) {
			resourceType := MapEndpointToResourceType(endpoint, defaultPermissions)
			assert.Equal(t, ResourceTypeOpen, resourceType,
				"Endpoint %s should map to ResourceTypeOpen", endpoint)
		})
	}
}

// TestMultipleHTTPMethods tests the capability checker with various HTTP methods
func TestMultipleHTTPMethods(t *testing.T) {
	checker := NewCapabilityChecker()

	// Define test cases for different HTTP methods and expected capability requirements
	testCases := []struct {
		method             string
		resourceType       ResourceType
		expectedCapability string
		isReadOperation    bool
	}{
		{http.MethodGet, ResourceTypeJob, "read:job", true},
		{http.MethodHead, ResourceTypeJob, "write:job", false}, // HEAD is not classified as read
		{http.MethodPost, ResourceTypeJob, "write:job", false},
		{http.MethodPut, ResourceTypeJob, "write:job", false},
		{http.MethodPatch, ResourceTypeJob, "write:job", false},
		{http.MethodDelete, ResourceTypeJob, "write:job", false},
		{http.MethodOptions, ResourceTypeJob, "write:job", false},
	}

	for _, tc := range testCases {
		t.Run(tc.method, func(t *testing.T) {
			capability := checker.GetRequiredCapability(tc.resourceType, tc.method)
			assert.Equal(t, tc.expectedCapability, capability,
				"HTTP method %s for %s should require %s",
				tc.method, tc.resourceType, tc.expectedCapability)
		})
	}
}

// TestEmptyOrInvalidEndpointPatterns tests handling of empty or invalid endpoint patterns
func TestEmptyOrInvalidEndpointPatterns(t *testing.T) {
	// Test with a map containing some edge cases
	endpointPermissions := map[string]string{
		"":              "invalid", // Empty endpoint
		"/":             "root",
		"/api/v1/":      "api",
		"/api/v1/valid": "valid",
	}

	// Test a valid endpoint
	validPath := "/api/v1/valid"
	resourceType := MapEndpointToResourceType(validPath, endpointPermissions)
	assert.Equal(t, ResourceType("valid"), resourceType)

	// Test with empty path - should return empty string as the function checks for path != ""
	emptyPath := ""
	resourceTypeEmpty := MapEndpointToResourceType(emptyPath, endpointPermissions)
	assert.Equal(t, ResourceType(""), resourceTypeEmpty, "Empty path should result in empty resource type")

	// Test with root path - should match the "/" endpoint
	rootPath := "/"
	resourceTypeRoot := MapEndpointToResourceType(rootPath, endpointPermissions)
	assert.Equal(t, ResourceType("root"), resourceTypeRoot, "Root path should match root endpoint")
}

// TestMissingCapability tests what happens when an endpoint doesn't have a corresponding capability defined
func TestMissingCapability(t *testing.T) {
	// Create a limited permissions map without some common endpoints
	limitedPermissions := map[string]string{
		"/api/v1/agent": "agent",
		"/api/v1/jobs":  "job",
	}

	// Test an endpoint that doesn't exist in the map
	missingPath := "/api/v1/unknown/endpoint"
	resourceType := MapEndpointToResourceType(missingPath, limitedPermissions)
	assert.Equal(t, ResourceType(""), resourceType, "Undefined endpoint should return empty resource type")

	// Test partial match that doesn't match any prefix
	partialPath := "/api/v2/agent"
	partialResourceType := MapEndpointToResourceType(partialPath, limitedPermissions)
	assert.Equal(t, ResourceType(""), partialResourceType, "Endpoint with unmatched prefix should return empty resource type")

	// Test what happens when we have an empty permissions map
	emptyPermissions := map[string]string{}
	emptyMapResourceType := MapEndpointToResourceType("/api/v1/agent", emptyPermissions)
	assert.Equal(t, ResourceType(""), emptyMapResourceType, "Any endpoint with empty permissions map should return empty resource type")

	// Test behavior with CheckUserAccess when resource type is empty
	checker := NewCapabilityChecker()
	user := types.AuthUser{
		Alias: "test_user",
		Capabilities: []types.Capability{
			{Actions: []string{"*"}}, // Even with all permissions
		},
	}

	// When a resource type is not mapped (resulting in empty resource type),
	// GetRequiredCapability will return write:node as default
	req := httptest.NewRequest(http.MethodGet, "/api/v1/unknown/endpoint", nil)
	hasAccess, requiredCapability := checker.CheckUserAccess(user, "", req)
	assert.Equal(t, "write:node", requiredCapability, "Unknown resource type should default to requiring write:node")
	assert.True(t, hasAccess, "User with * capability should have access even to unknown resources")

	// Test with a user without the required capability
	limitedUser := types.AuthUser{
		Alias: "limited_user",
		Capabilities: []types.Capability{
			{Actions: []string{"read:job"}}, // Limited permissions
		},
	}
	limitedHasAccess, _ := checker.CheckUserAccess(limitedUser, "", req)
	assert.False(t, limitedHasAccess, "User without write:node capability should not have access to unknown resources")
}

// TestHasRequiredCapabilityWithUndefinedCapabilities specifically tests the HasRequiredCapability function
// with various scenarios involving undefined or missing capabilities
func TestHasRequiredCapabilityWithUndefinedCapabilities(t *testing.T) {
	checker := NewCapabilityChecker()

	t.Run("Capability Not Present In User", func(t *testing.T) {
		// User with some defined capabilities but missing others
		user := types.AuthUser{
			Alias: "specific_user",
			Capabilities: []types.Capability{
				{Actions: []string{"read:job", "write:job"}},
			},
		}

		// Test capabilities the user doesn't have
		assert.False(t, checker.HasRequiredCapability(user, "read:node"),
			"User should not have read:node capability")
		assert.False(t, checker.HasRequiredCapability(user, "write:node"),
			"User should not have write:node capability")
		assert.False(t, checker.HasRequiredCapability(user, "read:agent"),
			"User should not have read:agent capability")
	})

	t.Run("Non-Standard Capability Format", func(t *testing.T) {
		// User with standard formatted capabilities
		user := types.AuthUser{
			Alias: "standard_user",
			Capabilities: []types.Capability{
				{Actions: []string{"read:job", "write:*"}},
			},
		}

		// Test with capabilities that don't follow the read:/write: pattern
		// Since these don't start with "read:", they're treated as non-read operations
		// and will match with write:* wildcard
		assert.True(t, checker.HasRequiredCapability(user, "admin:system"),
			"User with write:* should have admin:system capability")
		assert.True(t, checker.HasRequiredCapability(user, "manage:users"),
			"User with write:* should have manage:users capability")

		// Test with a user that only has read permissions
		readOnlyUser := types.AuthUser{
			Alias: "read_only_user",
			Capabilities: []types.Capability{
				{Actions: []string{"read:*"}},
			},
		}

		// Non-standard capabilities should NOT match read:* wildcard
		assert.False(t, checker.HasRequiredCapability(readOnlyUser, "admin:system"),
			"User with only read:* should NOT have admin:system capability")
		assert.False(t, checker.HasRequiredCapability(readOnlyUser, "manage:users"),
			"User with only read:* should NOT have manage:users capability")

		// Custom capability that doesn't start with read: but we have write:* wildcard
		assert.True(t, checker.HasRequiredCapability(user, "write:custom"),
			"User with write:* should have write:custom capability")
		assert.False(t, checker.HasRequiredCapability(user, "read:custom"),
			"User without read:custom should not have that capability")
	})

	t.Run("Wildcard vs Empty/Malformed Capabilities", func(t *testing.T) {
		// User with wildcard capability
		wildcardUser := types.AuthUser{
			Alias: "wildcard_user",
			Capabilities: []types.Capability{
				{Actions: []string{"read:*"}},
			},
		}

		// Test wildcard matching against different formats
		assert.True(t, checker.HasRequiredCapability(wildcardUser, "read:anything"),
			"User with read:* should have read:anything capability")
		assert.False(t, checker.HasRequiredCapability(wildcardUser, "write:anything"),
			"User with read:* should not have write:anything capability")

		// Test with empty capability string
		assert.False(t, checker.HasRequiredCapability(wildcardUser, ""),
			"User should not have empty string capability")

		// Test with malformed capability (no colon)
		assert.False(t, checker.HasRequiredCapability(wildcardUser, "readnode"),
			"User should not have malformed capability without colon")
	})

	t.Run("Index Out Of Range Prevention", func(t *testing.T) {
		// This test ensures the code doesn't panic with index out of range errors
		user := types.AuthUser{
			Alias: "test_user",
			Capabilities: []types.Capability{
				{Actions: []string{"read:*", "write:job"}},
			},
		}

		// Test with a short capability string that might cause index out of range
		// when the code checks if the required capability starts with "read"
		short := "r"
		assert.False(t, checker.HasRequiredCapability(user, short),
			"User should not have single-character capability")

		// Test with capability that is exactly 4 characters (edge case for the read: check)
		fourChar := "read"
		assert.False(t, checker.HasRequiredCapability(user, fourChar),
			"User should not have capability without resource part")
	})

	t.Run("Empty Capability Guard", func(t *testing.T) {
		// Test the guard against empty capabilities
		// This is important for security and preventing index out of range errors

		// User with universal access
		universalUser := types.AuthUser{
			Alias: "admin_user",
			Capabilities: []types.Capability{
				{Actions: []string{"*"}},
			},
		}

		// Even with universal access, empty capability should be denied
		assert.False(t, checker.HasRequiredCapability(universalUser, ""),
			"Empty capability string should always be denied, even for admin users")

		// User with specific capabilities
		specificUser := types.AuthUser{
			Alias: "specific_user",
			Capabilities: []types.Capability{
				{Actions: []string{"read:*", "write:*"}},
			},
		}

		// Empty capability should be denied
		assert.False(t, checker.HasRequiredCapability(specificUser, ""),
			"Empty capability string should always be denied")

		// User with no capabilities
		noCapabilitiesUser := types.AuthUser{
			Alias:        "no_capabilities_user",
			Capabilities: []types.Capability{},
		}

		// Empty capability should be denied
		assert.False(t, checker.HasRequiredCapability(noCapabilitiesUser, ""),
			"Empty capability string should always be denied, even when user has no capabilities")
	})
}

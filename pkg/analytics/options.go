package analytics

import (
	"os"
	"runtime"
	"strings"

	"github.com/Masterminds/semver"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// Option is a function that configures ResourceAttributes.
// Options are used with Setup to configure analytics.
type Option func(*ResourceAttributes)

// WithNodeID sets the hashed node ID in the resource attributes.
// The node ID is hashed for privacy before being stored.
//
// Parameters:
//   - id: The raw node ID to hash and store
func WithNodeID(id string) Option {
	return func(r *ResourceAttributes) {
		r.NodeIDHash = hashString(id)
	}
}

// WithNodeType sets the node type based on its roles.
// The node type can be "hybrid", "orchestrator", or "compute",
// depending on the combination of requester and compute roles.
//
// Parameters:
//   - isRequester: Whether this node acts as a requester/orchestrator
//   - isCompute: Whether this node acts as a compute provider
func WithNodeType(isRequester, isCompute bool) Option {
	return func(r *ResourceAttributes) {
		var typ string
		if isRequester && isCompute {
			typ = "hybrid"
		} else if isRequester {
			typ = "orchestrator"
		} else if isCompute {
			typ = "compute"
		}
		r.NodeType = typ
	}
}

// WithInstallationID sets the installation ID.
// The installation ID is a persistent identifier for a particular
// installation of the software, preserved across restarts.
//
// Parameters:
//   - id: The installation ID
func WithInstallationID(id string) Option {
	return func(r *ResourceAttributes) {
		if id != "" {
			r.InstallationID = id
		}
	}
}

// WithInstanceID sets the instance ID.
// The instance ID identifies a specific running instance of the node
// Preserved across restarts.
//
// Parameters:
//   - id: The instance ID
func WithInstanceID(id string) Option {
	return func(r *ResourceAttributes) {
		if id != "" {
			r.InstanceID = id
		}
	}
}

// WithVersion sets the node version from build information.
// This attempts to parse the version as a semantic version.
// If parsing fails, it uses the raw version string.
//
// Parameters:
//   - bv: Build version information
func WithVersion(bv *models.BuildVersionInfo) Option {
	return func(r *ResourceAttributes) {
		v, err := semver.NewVersion(bv.GitVersion)
		if err != nil {
			// use the version populated via the `ldflags` flag.
			r.NodeVersion = bv.GitVersion
		} else {
			r.NodeVersion = v.String()
		}
	}
}

// WithSystemInfo adds system information to the resource attributes.
// This includes OS type, architecture, and environment detection
// (Docker or local).
//
// This option reads environment variables and performs system checks,
// so it should be used with care in performance-sensitive contexts.
func WithSystemInfo() Option {
	return func(r *ResourceAttributes) {
		// Set OS information
		r.OSType = runtime.GOOS
		r.OSArch = runtime.GOARCH

		// Determine environment type
		if detectDockerEnvironment() {
			r.Environment = EnvDockerVal
		} else {
			r.Environment = EnvLocalVal
		}
	}
}

// detectDockerEnvironment checks if the current process is running
// inside a Docker container by looking for container-specific markers.
//
// Returns true if running in Docker, false otherwise.
func detectDockerEnvironment() bool {
	// Check for .dockerenv file (the simplest and most reliable indicator)
	_, err := os.Stat("/.dockerenv")
	if err == nil {
		return true
	}

	// Check for docker cgroup as a fallback method
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil && (os.IsNotExist(err) || os.IsPermission(err)) {
		return false
	}

	// If we could read the file and it's not empty, check for docker markers
	if err == nil && len(data) > 0 {
		content := string(data)
		// Look for Docker-specific markers in cgroup content
		dockerMarkers := []string{"/docker/", "/docker-", ".scope"}
		for _, marker := range dockerMarkers {
			if strings.Contains(content, marker) {
				return true
			}
		}
	}
	return false
}

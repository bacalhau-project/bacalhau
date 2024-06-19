package apimodels

import (
	"net/http"

	"github.com/Masterminds/semver"

	"github.com/bacalhau-project/bacalhau/pkg/version"
)

// GetClientVersion extracts the client version from the `X-Bacalhau-Git-Version` header in the request.
// If the header is present and the version string can be parsed, it returns the parsed version.
// If the header is missing or the version string cannot be parsed, it returns version.Unknown.
func GetClientVersion(req *http.Request) *semver.Version {
	if clientVerStr := req.Header.Get(HTTPHeaderBacalhauGitVersion); clientVerStr != "" {
		clientVersion, err := semver.NewVersion(clientVerStr)
		if err != nil {
			return version.Unknown
		}
		return clientVersion
	}
	return version.Unknown
}

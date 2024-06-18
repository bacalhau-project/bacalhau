package apimodels

import (
	"net/http"

	"github.com/Masterminds/semver"
)

var UnknownVersion = semver.MustParse("v0.0.0")

func GetClientVersion(req *http.Request) *semver.Version {
	if clientVerStr := req.Header.Get(HTTPHeaderBacalhauGitVersion); clientVerStr != "" {
		clientVersion, err := semver.NewVersion(clientVerStr)
		if err != nil {
			return UnknownVersion
		}
		return clientVersion
	}
	return UnknownVersion
}

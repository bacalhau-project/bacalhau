// Package version provides information about what Bacalhau was built from.
//
// The bulk of the information comes from the debug.BuildInfo struct which gets automatically populated when building a
// binary as a Go module (`go build .` vs `go build main.go`) with Go 1.18+. It contains various things such as VCS
// information or dependencies.
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/Masterminds/semver"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
)

var (
	// GITVERSION is the Git tag that Bacalhau was built from. This is expected to be populated via the `ldflags` flag,
	// at least until https://github.com/golang/go/issues/50603 is fixed. The value shown here will be used when the
	// value isn't provided by ldflags.
	//
	// A good article on how to use buildflags is
	// https://www.digitalocean.com/community/tutorials/using-ldflags-to-set-version-information-for-go-applications.
	GITVERSION = "v0.0.0-xxxxxxx"
)

// Get returns the overall codebase version. It's for detecting what code a binary was built from.
func Get() *model.BuildVersionInfo {
	revision, revisionTime, err := getBuildInformation()
	if err != nil {
		log.Fatal().Err(err).Msg("Could not build client information")
	}

	s, err := semver.NewVersion(GITVERSION)
	if err != nil {
		log.Fatal().Msgf("Could not parse GITVERSION during build - %s", GITVERSION)
	}

	versionInfo := &model.BuildVersionInfo{
		Major:      strconv.FormatInt(s.Major(), 10), //nolint:gomnd // base10, magic number appropriate
		Minor:      strconv.FormatInt(s.Minor(), 10), //nolint:gomnd // base10, magic number appropriate
		GitVersion: GITVERSION,
		GitCommit:  revision,
		BuildDate:  revisionTime,
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
	}

	return versionInfo
}

func getBuildInformation() (string, time.Time, error) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", time.Time{}, fmt.Errorf("binary not built as a Go module")
	}

	// Fallback values used when _not_ built as a Go module, such as when running tests.
	revision := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	revisionTimeStr := "1970-01-01T00:00:00Z"

	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			revision = setting.Value
		} else if setting.Key == "vcs.time" {
			revisionTimeStr = setting.Value
		}
	}

	revisionTime, err := time.Parse(time.RFC3339Nano, revisionTimeStr)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("couldn't parse revision date: %w", err)
	}

	return revision, revisionTime, nil
}

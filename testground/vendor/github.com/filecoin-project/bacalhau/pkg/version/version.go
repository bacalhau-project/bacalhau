/*
Originally:

Copyright 2014 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package version

import (
	"strconv"
	"time"

	"github.com/Masterminds/semver"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
)

// Get returns the overall codebase version. It's for detecting
// what code a binary was built from.
func Get() *model.VersionInfo {
	// These variables typically come from -ldflags settings and in
	// their absence fallback to the settings in pkg/version/base.go

	versionInfo := &model.VersionInfo{}
	s, err := semver.NewVersion(GITVERSION)
	if err != nil {
		log.Fatal().Msgf("Could not parse GITVERSION during build - %s", GITVERSION)
	}
	versionInfo.GitVersion = GITVERSION
	versionInfo.Major = strconv.FormatInt(s.Major(), 10) //nolint:gomnd // base10, magic number appropriate
	versionInfo.Minor = strconv.FormatInt(s.Minor(), 10) //nolint:gomnd // base10, magic number appropriate
	versionInfo.GitCommit = GITCOMMIT
	buildDate, err := time.Parse("2006-01-02T15:04:05Z", BUILDDATE)
	if err != nil {
		log.Fatal().Msgf("Could not parse BUILDDATE during build - %s", GITVERSION)
	}

	versionInfo.BuildDate = buildDate
	versionInfo.GOOS = GOOS
	versionInfo.GOARCH = GOARCH

	return versionInfo
}

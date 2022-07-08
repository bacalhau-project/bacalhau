/* Copyright: TBD */

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

// Base version information.
//
// This is the fallback data used when version information from git is not
// provided via go ldflags. It provides an approximation of the Kubernetes
// version for ad-hoc builds (e.g. `go build`) that cannot get the version
// information from git.

// Good article on how to use buildflags
// https://www.digitalocean.com/community/tutorials/using-ldflags-to-set-version-information-for-go-applications

var (
	GITVERSION string = "v0.0.0-xxxxxxx"
	GITCOMMIT  string = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	BUILDDATE  string = "1970-01-01T00:00:00Z" // build date in ISO8601 format, output of $(date -u +'%Y-%m-%dT%H:%M:%SZ')
	GOOS       string = "aaaaa"
	GOARCH     string = "bbb00"
)

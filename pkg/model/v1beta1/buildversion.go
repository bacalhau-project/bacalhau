package v1beta1

import (
	"time"
)

// BuildVersionInfo is the version of a Bacalhau binary (either client or server)
type BuildVersionInfo struct {
	// Client Version: version.Info{Major:"1", Minor:"24", GitVersion:"v1.24.0",
	// GitCommit:"4ce5a8954017644c5420bae81d72b09b735c21f0", GitTreeState:"clean",
	// BuildDate:"2022-05-03T13:46:05Z", GoVersion:"go1.18.1", Compiler:"gc", Platform:"darwin/arm64"}

	Major      string    `json:"major,omitempty" example:"0"`
	Minor      string    `json:"minor,omitempty" example:"3"`
	GitVersion string    `json:"gitversion" example:"v0.3.12"`
	GitCommit  string    `json:"gitcommit" example:"d612b63108f2b5ce1ab2b9e02444eb1dac1d922d"`
	BuildDate  time.Time `json:"builddate" example:"2022-11-16T14:03:31Z"`
	GOOS       string    `json:"goos" example:"linux"`
	GOARCH     string    `json:"goarch" example:"amd64"`
}

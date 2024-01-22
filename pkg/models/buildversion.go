package models

import (
	"time"
)

// BuildVersionInfo is the version of a Bacalhau binary (either client or server)
type BuildVersionInfo struct {
	Major      string    `json:"Major,omitempty" example:"0"`
	Minor      string    `json:"Minor,omitempty" example:"3"`
	GitVersion string    `json:"GitVersion" example:"v0.3.12"`
	GitCommit  string    `json:"GitCommit" example:"d612b63108f2b5ce1ab2b9e02444eb1dac1d922d"`
	BuildDate  time.Time `json:"BuildDate" example:"2022-11-16T14:03:31Z"`
	GOOS       string    `json:"GOOS" example:"linux"`
	GOARCH     string    `json:"GOARCH" example:"amd64"`
}

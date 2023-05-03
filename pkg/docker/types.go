package docker

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type imageResolverFunc func(context.Context, string, bool, config.DockerCredentials) (*ImageManifest, error)

type ImageManifest struct {
	digest    string
	platforms []v1.Platform
}

package docker

import v1 "github.com/opencontainers/image-spec/specs-go/v1"

type ImageManifest struct {
	digest    string
	platforms []v1.Platform
}

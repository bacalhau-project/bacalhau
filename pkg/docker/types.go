package docker

import (
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type ImageManifest struct {
	// We only ever expect the digest to be the `algorithm:hash`
	Digest    digest.Digest
	Platforms []v1.Platform
}

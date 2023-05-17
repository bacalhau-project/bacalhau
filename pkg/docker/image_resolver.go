package docker

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/cache"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/rs/zerolog/log"
)

type ImageResolver struct {
	source   *ImageID
	resolved string
}

func NewImageResolver(orig *ImageID) *ImageResolver {
	return &ImageResolver{source: orig}
}

func (r *ImageResolver) Resolve(ctx context.Context, resolver imageResolverFunc, tagCache cache.Cache[string]) error {
	client, err := NewDockerClient()
	if err != nil || !client.IsInstalled(ctx) {
		return fmt.Errorf("docker is not installed")
	}

	if r.source.HasDigest() {
		r.resolved = r.source.String()
		return nil
	}

	// Attempt to find a digest in the local cache so that we don't need to make a
	// call to docker to ask.
	cachedDigest, found := tagCache.Get(r.source.String())
	if found {
		r.resolved = cachedDigest
		return nil
	}

	credentials := config.GetDockerCredentials()
	manifest, err := resolver(ctx, r.source.String(), credentials)
	if err != nil {
		log.Ctx(ctx).Error().
			Err(err).
			Stringer("Image", r.source).
			Msg("failed to get image digest")
		return err
	}

	cloned, _ := NewImageID(r.source.String())
	cloned.tag = DigestTag(manifest.digest)
	r.resolved = cloned.String()

	// Save a copy of the digest in the local cache for a set period of time
	// so that we can avoid an API call next time around
	cacheDuration := r.source.tag.CacheDuration()
	_ = tagCache.Set(r.source.String(), r.resolved, 1, cacheDuration)

	return nil
}

func (r *ImageResolver) Digest() string {
	return r.resolved
}

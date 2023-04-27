package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/rs/zerolog/log"
)

type imageResolverFunc func(context.Context, string, bool, config.DockerCredentials) (*ImageManifest, error)

// ResolveImageID will take the provided image identifier and a resolver,
// and attempt to provide a version of the image id containing the digest
// instead.
func ResolveImageID(ctx context.Context, img ImageID, resolver imageResolverFunc) (ImageID, error) {
	if img.HasDigest() {
		return img, nil
	}

	// TODO: Look up i.String() in cache to see if we already have a digest for it

	credentials := config.GetDockerCredentials()
	manifest, err := resolver(ctx, img.String(), false, credentials)
	if err != nil {
		log.Ctx(ctx).Error().
			Err(err).
			Str("Image", img.String()).
			Msg("failed to get image digest")
		return img, err
	}

	if !strings.HasPrefix(manifest.digest, "sha256") {
		// Need to make sure digest is complete
		manifest.digest = fmt.Sprintf("sha256:%s", manifest.digest)
	}

	result := img
	result.tag = DigestTag(manifest.digest)
	return result, nil
}

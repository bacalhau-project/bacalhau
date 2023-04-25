package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/rs/zerolog/log"
)

type imageResolverFunc func(context.Context, string, config.DockerCredentials) (string, error)

// ResolveImageID will take the provided image identifier and a resolver,
// and attempt to provide a version of the image id containing the digest
// instead.
func ResolveImageID(ctx context.Context, i ImageID, resolver imageResolverFunc) (ImageID, error) {
	if i.HasDigest() {
		return i, nil
	}

	// TODO: Look up i.String() in cache to see if we already have a digest for it

	credentials := config.GetDockerCredentials()
	digest, err := resolver(ctx, i.String(), credentials)
	if err != nil {
		log.Ctx(ctx).Error().
			Err(err).
			Str("Image", i.String()).
			Msg("failed to get image digest")
		return i, err
	}

	if !strings.HasPrefix(digest, "sha256") {
		// Need to make sure digest is complete
		digest = fmt.Sprintf("sha256:%s", digest)
	}

	result := i
	result.tag = DigestTag(digest)
	return result, nil
}

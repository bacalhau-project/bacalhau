package jobtransform

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
)

func DockerImageDigest() Transformer {
	return func(ctx context.Context, j *model.Job) (modified bool, err error) {
		client, err := docker.NewDockerClient()
		if err != nil || !client.IsInstalled(ctx) {
			// Return a noop if docker is not installed as it means we
			// won't be able to find digests for images in the requester
			return false, nil
		}

		if j.Spec.Engine != model.EngineDocker {
			return false, nil
		}

		image, err := docker.NewImageID(j.Spec.Docker.Image)
		if err != nil {
			return false, nil
		}

		resolver := docker.NewImageResolver(image)
		err = resolver.Resolve(ctx, client.ImageDistribution, docker.DockerTagCache)
		if err != nil {
			log.Ctx(ctx).Debug().
				Str("Image", image.String()).
				Msg("failed to find digest for image")
			return false, nil
		}

		j.Spec.Docker.Image = resolver.Digest()
		log.Ctx(ctx).Debug().
			Str("OldImage", image.String()).
			Str("NewImage", j.Spec.Docker.Image).
			Msg("updated docker image with digest")

		return true, nil
	}
}

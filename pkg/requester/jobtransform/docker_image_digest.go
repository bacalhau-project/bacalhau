package jobtransform

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
)

func DockerImageDigest() Transformer {
	client, err := docker.NewDockerClient()

	// With no context available we are happy to accept we can't cancel
	// this local IPC call to the docker daemon
	if err != nil || !client.IsInstalled(context.Background()) {
		// Return a noop if docker is not installed as it means we
		// won't be able to find digests for images in the requester
		return func(context.Context, *model.Job) (bool, error) {
			return false, nil
		}
	}

	return func(ctx context.Context, j *model.Job) (modified bool, err error) {
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
				Stringer("Image", image).
				Msg("failed to find digest for image")
			return false, nil
		}

		j.Spec.Docker.Image = resolver.Digest()
		log.Ctx(ctx).Debug().
			Stringer("OldImage", image).
			Str("NewImage", j.Spec.Docker.Image).
			Msg("updated docker image with digest")

		return true, nil
	}
}

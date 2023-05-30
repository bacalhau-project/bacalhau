package jobtransform

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	spec_docker "github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/docker"

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
		if j.Spec.Engine.Schema != spec_docker.EngineType {
			return false, nil
		}

		dockerEngine, err := spec_docker.Decode(j.Spec.Engine)
		if err != nil {
			return false, err
		}

		image, err := docker.NewImageID(dockerEngine.Image)
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

		j.Spec.Engine, err = spec_docker.Mutate(j.Spec.Engine, spec_docker.WithImage(resolver.Digest()))
		if err != nil {
			return false, err
		}

		log.Ctx(ctx).Debug().
			Stringer("OldImage", image).
			Str("NewImage", resolver.Digest()).
			Msg("updated docker image with digest")

		return true, nil
	}
}

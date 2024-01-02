package jobtransform

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/model"
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
		if j.Spec.EngineSpec.Engine() != model.EngineDocker {
			return false, nil
		}

		dockerEngine, err := model.DecodeEngineSpec[model.DockerEngineSpec](j.Spec.EngineSpec)
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

		// TODO(forrest): [review] unsure if this is the best DevEx for mutating an engine.
		j.Spec.EngineSpec = model.NewDockerEngineBuilder(resolver.Digest()).
			WithParameters(dockerEngine.Parameters...).
			WithEntrypoint(dockerEngine.Entrypoint...).
			WithEnvironmentVariables(dockerEngine.EnvironmentVariables...).
			WithWorkingDirectory(dockerEngine.WorkingDirectory).
			Build()

		log.Ctx(ctx).Debug().
			Stringer("OldImage", image).
			Str("NewImage", resolver.Digest()).
			Msg("updated docker image with digest")

		return true, nil
	}
}

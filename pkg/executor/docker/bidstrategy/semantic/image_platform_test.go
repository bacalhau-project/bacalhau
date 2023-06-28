//go:build integration || !unit

package semantic_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/cache"
	"github.com/bacalhau-project/bacalhau/pkg/cache/fake"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor/docker/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

func jobForDockerImage(imageID string) model.Job {
	return model.Job{
		Spec: model.Spec{
			EngineDeprecated: model.EngineDocker,
			EngineSpec: model.DockerEngineSpec{
				Image: imageID,
			}.AsEngineSpec(),
		},
	}
}

func TestBidsBasedOnImagePlatform(t *testing.T) {
	docker.MustHaveDocker(t)

	client, err := docker.NewDockerClient()
	require.NoError(t, err)

	strategy := semantic.NewImagePlatformBidStrategy(client)

	t.Run("positive response for supported architecture", func(t *testing.T) {
		response, err := strategy.ShouldBid(context.Background(), bidstrategy.BidStrategyRequest{
			Job: jobForDockerImage("ubuntu"),
		})

		require.NoError(t, err)
		require.Equal(t, true, response.ShouldBid)
	})

	t.Run("negative response for unsupported architecture", func(t *testing.T) {
		response, err := strategy.ShouldBid(context.Background(), bidstrategy.BidStrategyRequest{
			Job: jobForDockerImage("mcr.microsoft.com/windows:ltsc2019"),
		})

		require.NoError(t, err)
		require.Equal(t, false, response.ShouldBid)
	})

	t.Run("cached manifest response for duplicate call", func(t *testing.T) {

		previousCache := semantic.ManifestCache

		var fc *fake.FakeCache[docker.ImageManifest] = fake.NewFakeCache[docker.ImageManifest]()
		var cc cache.Cache[docker.ImageManifest] = fc
		semantic.ManifestCache = &cc

		response, err := strategy.ShouldBid(context.Background(), bidstrategy.BidStrategyRequest{
			Job: jobForDockerImage("ubuntu:latest"),
		})

		require.NoError(t, err)
		require.Equal(t, true, response.ShouldBid)

		// Second time we expect should be cached
		response, err = strategy.ShouldBid(context.Background(), bidstrategy.BidStrategyRequest{
			Job: jobForDockerImage("ubuntu:latest"),
		})

		require.NoError(t, err)
		require.Equal(t, true, response.ShouldBid)

		// We expect the cache to contain one item,
		// and have called Set twice, and Get twice with
		// one successful and one failed lookup.
		require.Equal(t, 1, fc.ItemCount())
		require.Equal(t, 2, fc.SetCalls)
		require.Equal(t, 2, fc.GetCalls)
		require.Equal(t, 1, fc.SuccessfulGetCalls)

		// Reset the cache to the default impl
		semantic.ManifestCache = previousCache
	})
}

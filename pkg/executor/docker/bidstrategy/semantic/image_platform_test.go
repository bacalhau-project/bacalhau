//go:build integration || !unit

package semantic_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	dockermodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/cache"
	"github.com/bacalhau-project/bacalhau/pkg/cache/fake"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor/docker/bidstrategy/semantic"
)

func jobForDockerImage(t testing.TB, imageID string) models.Job {
	job := mock.Job()
	var err error
	job.Task().Engine, err = dockermodels.NewDockerEngineBuilder(imageID).Build()
	require.NoError(t, err)
	return *job
}

func TestBidsBasedOnImagePlatform(t *testing.T) {
	docker.MustHaveDocker(t)

	client, err := docker.NewDockerClient()
	require.NoError(t, err)

	testConfig, err := config.NewTestConfig()
	require.NoError(t, err)

	strategy := semantic.NewImagePlatformBidStrategy(client,
		types.DockerManifestCache{
			Size:    testConfig.Engines.Types.Docker.ManifestCache.Size,
			TTL:     testConfig.Engines.Types.Docker.ManifestCache.TTL,
			Refresh: testConfig.Engines.Types.Docker.ManifestCache.Refresh,
		},
	)

	t.Run("positive response for supported architecture", func(t *testing.T) {
		response, err := strategy.ShouldBid(context.Background(), bidstrategy.BidStrategyRequest{
			Job: jobForDockerImage(t, "busybox:1.37.0"),
		})

		require.NoError(t, err)
		require.Equal(t, true, response.ShouldBid)
	})

	t.Run("negative response for unsupported architecture", func(t *testing.T) {
		response, err := strategy.ShouldBid(context.Background(), bidstrategy.BidStrategyRequest{
			Job: jobForDockerImage(t, "mcr.microsoft.com/windows:ltsc2019"),
		})

		require.NoError(t, err)
		require.Equal(t, false, response.ShouldBid)
	})

	t.Run("cached manifest response for duplicate call", func(t *testing.T) {

		previousCache := semantic.ManifestCache

		var fc *fake.FakeCache[docker.ImageManifest] = fake.NewFakeCache[docker.ImageManifest]()
		var cc cache.Cache[docker.ImageManifest] = fc
		semantic.ManifestCache = cc

		response, err := strategy.ShouldBid(context.Background(), bidstrategy.BidStrategyRequest{
			Job: jobForDockerImage(t, "busybox:latest"),
		})

		require.NoError(t, err)
		require.Equal(t, true, response.ShouldBid)

		// Second time we expect should be cached
		response, err = strategy.ShouldBid(context.Background(), bidstrategy.BidStrategyRequest{
			Job: jobForDockerImage(t, "busybox:latest"),
		})

		require.NoError(t, err)
		require.Equal(t, true, response.ShouldBid)

		// We expect the cache to contain one item,
		// and have called Set twice, and Get twice with
		// one successful and one failed lookup.
		require.Equal(t, 1, fc.ItemCount())
		require.Equal(t, 1, fc.SetCalls)
		require.Equal(t, 1, fc.FailedGetCalls)
		require.Equal(t, 2, fc.GetCalls)
		require.Equal(t, 1, fc.SuccessfulGetCalls)

		// Reset the cache to the default impl
		semantic.ManifestCache = previousCache
	})
}

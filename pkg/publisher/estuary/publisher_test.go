//go:build integration

package estuary

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/stretchr/testify/require"
)

func getPublisherWithGoodConfig(t *testing.T) publisher.Publisher {
	apiKey, isSet := os.LookupEnv("ESTUARY_API_KEY")
	if !isSet {
		t.Skip("No ESTUARY_API_KEY set")
	}

	return NewEstuaryPublisher(EstuaryPublisherConfig{APIKey: apiKey})
}

func getPublisherWithErrorConfig(*testing.T) publisher.Publisher {
	return NewEstuaryPublisher(EstuaryPublisherConfig{APIKey: "TEST"})
}

func TestIsInstalled(t *testing.T) {
	publisher := getPublisherWithGoodConfig(t)
	isInstalled, err := publisher.IsInstalled(context.Background())
	require.True(t, isInstalled)
	require.NoError(t, err)
}

func TestNotInstalledWithBadKey(t *testing.T) {
	publisher := getPublisherWithErrorConfig(t)
	isInstalled, err := publisher.IsInstalled(context.Background())
	require.False(t, isInstalled)
	require.NoError(t, err)
}

func TestUpload(t *testing.T) {
	tempDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tempDir, "hello.txt"), []byte("hello, world!"), os.ModePerm)
	require.NoError(t, err)

	publisher := getPublisherWithGoodConfig(t)
	spec, err := publisher.PublishShardResult(
		context.Background(),
		model.JobShard{
			Job:   model.NewJob(),
			Index: 0,
		},
		"host",
		tempDir,
	)
	require.NoError(t, err)
	require.Equal(t, spec.StorageSource, model.StorageSourceEstuary)
	require.NotEmpty(t, spec.CID)
}

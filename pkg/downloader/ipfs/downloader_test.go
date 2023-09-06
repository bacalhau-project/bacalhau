//go:build integration || !unit

package ipfs

import (
	"context"
	"crypto/rand"
	"path/filepath"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/configenv"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
	"github.com/stretchr/testify/require"
)

func randomText(t *testing.T) []byte {
	text := make([]byte, 256)
	n, err := rand.Read(text)
	require.NoError(t, err)
	require.Equal(t, 256, n)
	return text
}

func TestIPFSDownload(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	cm := system.NewCleanupManager()
	defer cm.Cleanup(ctx)
	defer cancel()

	server, err := ipfs.NewNodeWithConfig(ctx, cm, types.IpfsConfig{PrivateInternal: true})
	require.NoError(t, err)

	text := randomText(t)
	cid, err := ipfs.AddTextToNodes(ctx, text, server.Client())
	require.NoError(t, err)

	swarm, err := server.SwarmAddresses()
	require.NoError(t, err)

	cfg := configenv.Testing
	cfg.Node.IPFS.SwarmAddresses = swarm
	config.Set(cfg)

	outputDir := t.TempDir()
	downloader := NewIPFSDownloader(cm, &model.DownloaderSettings{
		Timeout:   time.Minute,
		OutputDir: outputDir,
	})

	err = downloader.FetchResult(ctx, model.DownloadItem{
		CID:        cid,
		SourceType: model.StorageSourceIPFS,
		Target:     filepath.Join(outputDir, "output.txt"),
	})
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(outputDir, "output.txt"))
}

//go:build integration || !unit

package ipfs

import (
	"context"
	"crypto/rand"
	"os"
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

const testSwarmKey = "/key/swarm/psk/1.0.0/\n/base16/\n463ff859373f8e89dd23e7d5429864c84283d961148dc311d120534780549ec3\n"

func writeSwarmKey(t *testing.T) string {
	file, err := os.CreateTemp(t.TempDir(), "swarm.key")
	require.NoError(t, err)
	defer closer.CloseWithLogOnError(file.Name(), file)

	n, err := file.WriteString(testSwarmKey)
	require.NoError(t, err)
	require.Equal(t, len(testSwarmKey), n)

	return file.Name()
}

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

func TestDownloadFromPrivateSwarm(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	cm := system.NewCleanupManager()
	defer cm.Cleanup(ctx)
	defer cancel()

	cfg := configenv.Testing
	cfg.Node.IPFS.PrivateInternal = true
	cfg.Node.IPFS.SwarmKeyPath = writeSwarmKey(t)
	config.Set(cfg)

	server, err := ipfs.NewNodeWithConfig(ctx, cm, cfg.Node.IPFS)
	require.NoError(t, err)

	text := randomText(t)
	cid, err := ipfs.AddTextToNodes(ctx, text, server.Client())
	require.NoError(t, err)

	swarm, err := server.SwarmAddresses()
	require.NoError(t, err)

	cfg.Node.IPFS.SwarmAddresses = swarm
	config.Set(cfg)

	t.Run("download success with swarm key", func(t *testing.T) {
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
	})

	cfg = configenv.Testing
	cfg.Node.IPFS.SwarmKeyPath = ""
	cfg.Node.IPFS.SwarmAddresses = swarm
	config.Set(cfg)

	t.Run("download failure without swarm key", func(t *testing.T) {
		outputDir := t.TempDir()
		downloader := NewIPFSDownloader(cm, &model.DownloaderSettings{
			Timeout:   10 * time.Second,
			OutputDir: outputDir,
		})

		err = downloader.FetchResult(ctx, model.DownloadItem{
			CID:        cid,
			SourceType: model.StorageSourceIPFS,
			Target:     filepath.Join(outputDir, "output.txt"),
		})
		require.Error(t, err)
		require.NoFileExists(t, filepath.Join(outputDir, "output.txt"))
	})
}

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
	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	ipfssource "github.com/bacalhau-project/bacalhau/pkg/storage/ipfs"
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
	connString := ipfs.MustHaveIPFS(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	cm := system.NewCleanupManager()
	defer cm.Cleanup(ctx)
	defer cancel()

	client, err := ipfs.NewClientUsingRemoteHandler(ctx, connString)
	require.NoError(t, err)

	text := randomText(t)
	cid, err := ipfs.AddTextToNodes(ctx, text, client)
	require.NoError(t, err)

	outputDir := t.TempDir()
	ipfsDownloader := NewIPFSDownloader(cm, client)

	resultPath, err := ipfsDownloader.FetchResult(ctx, downloader.DownloadItem{
		Result: &models.SpecConfig{
			Type: models.StorageSourceIPFS,
			Params: ipfssource.Source{
				CID: cid,
			}.ToMap(),
		},
		ParentPath: outputDir,
	})
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(outputDir, cid))
	require.Equal(t, filepath.Join(outputDir, cid), resultPath)
}

func TestDownloadFromPrivateSwarm(t *testing.T) {
	connString := ipfs.MustHaveIPFS(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	cm := system.NewCleanupManager()
	defer cm.Cleanup(ctx)
	defer cancel()

	cfg := configenv.Testing
	cfg.Node.IPFS.SwarmKeyPath = writeSwarmKey(t)
	config.Set(cfg)

	client, err := ipfs.NewClientUsingRemoteHandler(ctx, connString)
	require.NoError(t, err)

	text := randomText(t)
	cid, err := ipfs.AddTextToNodes(ctx, text, client)
	require.NoError(t, err)

	t.Run("download success with swarm key", func(t *testing.T) {
		outputDir := t.TempDir()
		ipfsDownloader := NewIPFSDownloader(cm, client)

		resultPath, err := ipfsDownloader.FetchResult(ctx, downloader.DownloadItem{
			Result: &models.SpecConfig{
				Type: models.StorageSourceIPFS,
				Params: ipfssource.Source{
					CID: cid,
				}.ToMap(),
			},
			ParentPath: outputDir,
		})

		require.NoError(t, err)
		require.FileExists(t, filepath.Join(outputDir, cid))
		require.Equal(t, filepath.Join(outputDir, cid), resultPath)
	})

	cfg = configenv.Testing
	cfg.Node.IPFS.SwarmKeyPath = ""
	cfg.Node.IPFS.SwarmAddresses = swarm
	config.Set(cfg)

	t.Run("download failure without swarm key", func(t *testing.T) {
		// This fails by timing out, but does so after 2minutes. We should give it
		// 5 seconds to find the file, which is plenty given success takes ms.
		cTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)

		outputDir := t.TempDir()
		ipfsDownloader := NewIPFSDownloader(cm, client)
		resultPath, err := ipfsDownloader.FetchResult(cTimeout, downloader.DownloadItem{
			Result: &models.SpecConfig{
				Type: models.StorageSourceIPFS,
				Params: ipfssource.Source{
					CID: cid,
				}.ToMap(),
			},
			ParentPath: outputDir,
		})
		cancel()

		require.Error(t, err)
		require.NoFileExists(t, filepath.Join(outputDir, cid))
		require.Equal(t, "", resultPath)
	})
}

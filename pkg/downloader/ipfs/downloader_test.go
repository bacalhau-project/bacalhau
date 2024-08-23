//go:build integration || !unit

package ipfs

import (
	"context"
	"crypto/rand"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	ipfssource "github.com/bacalhau-project/bacalhau/pkg/storage/ipfs"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
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
	defer cancel()

	_, cfg := setup.SetupBacalhauRepoForTesting(t)
	ipfsConnect := testutils.MustHaveIPFS(t, cfg)

	client, err := ipfs.NewClient(context.Background(), ipfsConnect)
	require.NoError(t, err)

	text := randomText(t)
	cid, err := ipfs.AddTextToNodes(ctx, text, *client)
	require.NoError(t, err)

	outputDir := t.TempDir()
	ipfsDownloader := NewIPFSDownloader(client)

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

//go:build !unit || integration

package estuary

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/stretchr/testify/require"
)

const testCID = "bafkreihhfsv64fxhjix43i66vue6ezcwews3eg6tacxar7mnkqrg5vn6pe"

func TestFetchResult(t *testing.T) {
	// create a temp directory for the downloaded file
	downloadDir, err := os.MkdirTemp("", "estuary-download-test")
	require.NoError(t, err)

	defer os.RemoveAll(downloadDir)

	// create a new Estuary downloader
	settings := &model.DownloaderSettings{
		Timeout: time.Second * 60,
	}
	downloader, err := NewEstuaryDownloader(settings)
	require.NoError(t, err)

	// create a PublishedResult with the test CID
	result := model.PublishedResult{
		Data: model.StorageSpec{
			StorageSource: model.StorageSourceEstuary,
			Name:          testCID,
			CID:           testCID,
			URL:           "https://api.estuary.tech/gw/ipfs/bafkreihhfsv64fxhjix43i66vue6ezcwews3eg6tacxar7mnkqrg5vn6pe",
		},
	}

	// call FetchResult to download the file
	err = downloader.FetchResult(context.Background(), result, downloadDir)
	require.NoError(t, err)

	// check that the file was downloaded to the correct location
	downloadedFile := fmt.Sprintf("%s/%s", downloadDir, testCID)
	if _, err := os.Stat(downloadedFile); os.IsNotExist(err) {
		t.Errorf("Expected file %s to be downloaded, but it does not exist", downloadedFile)
	}

	// check the content of the downloaded file
	data, err := os.ReadFile(downloadedFile)
	require.NoError(t, err)

	require.Equal(t, "Hello From Bacalhau\n", string(data))
}

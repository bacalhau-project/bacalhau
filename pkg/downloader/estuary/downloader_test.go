//go:build integration || !unit

package estuary

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/system"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/stretchr/testify/require"
)

const testCID = "bafkreihhfsv64fxhjix43i66vue6ezcwews3eg6tacxar7mnkqrg5vn6pe"
const testURL = "https://api.estuary.tech/gw/ipfs/bafkreihhfsv64fxhjix43i66vue6ezcwews3eg6tacxar7mnkqrg5vn6pe"

func TestFetchResult(t *testing.T) {
	// create a new Estuary downloader
	settings := &model.DownloaderSettings{
		Timeout: time.Second * 60,
	}
	cm := system.NewCleanupManager()
	downloader := NewEstuaryDownloader(cm, settings)

	tests := []struct {
		CID  string
		Name string
		URL  string
	}{
		{
			CID: testCID, Name: testCID, URL: testURL,
		},
		{
			CID: testCID, Name: "", URL: "",
		},
	}

	for _, ts := range tests {
		// create a temp directory for the downloaded file
		downloadDir, err := os.MkdirTemp("", "estuary-download-testData-*")
		require.NoError(t, err)
		downloadPath := filepath.Join(downloadDir, testCID)

		// create a PublishedResult with the test data
		result := model.PublishedResult{
			Data: model.StorageSpec{
				StorageSource: model.StorageSourceEstuary,
				Name:          ts.Name,
				CID:           ts.CID,
				URL:           ts.URL,
			},
		}

		// call FetchResult to download the file
		err = downloader.FetchResult(context.Background(), result, downloadPath)
		require.NoError(t, err)

		// check that the file was downloaded to the correct location
		if _, err := os.Stat(downloadPath); os.IsNotExist(err) {
			t.Errorf("Expected file %s to be downloaded, but it does not exist", downloadPath)
		}

		// check the content of the downloaded file
		data, err := os.ReadFile(downloadPath)
		require.NoError(t, err)

		require.Equal(t, "Hello From Bacalhau\n", string(data))

		err = os.RemoveAll(downloadDir)
		require.NoError(t, err)
	}
}

package urldownload

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type StorageSuite struct {
	suite.Suite
	RootCmd *cobra.Command
}

func TestStorageSuite(t *testing.T) {
	suite.Run(t, new(StorageSuite))
}

// Before all suite
func (s *StorageSuite) SetupSuite() {
}

// Before each test
func (s *StorageSuite) SetupTest() {
	require.NoError(s.T(), system.InitConfigForTesting())
}

func (s *StorageSuite) TearDownTest() {
}

func (s *StorageSuite) TearDownSuite() {

}

func (s *StorageSuite) TestNewStorageProvider() {
	cm := system.NewCleanupManager()

	sp, err := NewStorage(cm)
	require.NoError(s.T(), err, "failed to create storage provider")

	// is dir writable?
	fmt.Println(sp.LocalDir)
	f, err := os.Create(filepath.Join(sp.LocalDir, "data.txt"))
	require.NoError(s.T(), err, "failed to create file")

	_, err = f.WriteString("test\n")
	require.NoError(s.T(), err, "failed to write to file")

	f.Close()
	if sp.HTTPClient == nil {
		require.Fail(s.T(), "HTTPClient is nil")
	}
}

func (s *StorageSuite) TestHasStorageLocally() {
	cm := system.NewCleanupManager()
	ctx := context.Background()

	sp, err := NewStorage(cm)
	require.NoError(s.T(), err, "failed to create storage provider")

	spec := model.StorageSpec{
		StorageSource: model.StorageSourceURLDownload,
		URL:           "foo",
		Path:          "foo",
	}
	// files are not cached thus shall never return true
	locally, err := sp.HasStorageLocally(ctx, spec)
	require.NoError(s.T(), err, "failed to check if storage is locally available")

	if locally != false {
		require.Fail(s.T(), "storage should not be locally available")
	}
}

func (s *StorageSuite) TestPrepareStorage() {
	fileName := "testfile"
	testString := "Here's your data"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() == "/testfile" {
			w.Write([]byte(testString))
		}
	}))
	defer ts.Close()

	cm := system.NewCleanupManager()
	ctx := context.Background()
	sp, err := NewStorage(cm)
	require.NoError(s.T(), err, "failed to create storage provider")

	serverURL := ts.URL
	spec := model.StorageSpec{
		StorageSource: model.StorageSourceURLDownload,
		URL:           serverURL + "/testfile",
		Path:          "/foo",
	}

	volume, err := sp.PrepareStorage(ctx, spec)
	require.NoError(s.T(), err, "failed to prepare storage")

	file, err := os.Open(filepath.Join(volume.Source, fileName))
	require.NoError(s.T(), err, "failed to open file")

	defer func() {
		if err = file.Close(); err != nil {
			require.Fail(s.T(), "failed to close file")
		}
	}()

	content, err := ioutil.ReadAll(file)
	require.NoError(s.T(), err, "failed to read file")

	text := string(content)
	require.Equal(s.T(), testString, text, "content of file does not match")
}

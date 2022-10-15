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

func (s *StorageSuite) TestPrepareStorageURL() {
	fileName := "testfile.py"
	testString := "Here's your data"

	redirectCases := map[string]struct {
		redirect bool
	}{
		"no-redirect": {
			redirect: false,
		},
		"redirect": {
			redirect: true,
		},
	}

	filetypeCases := map[string]struct {
		fileName      string
		content       string
		valid         bool
		errorContains string
		errorMsg      string
	}{
		"Test-Valid": {fileName: fileName,
			content:       testString,
			valid:         true,
			errorContains: "",
			errorMsg:      "TYPE: Valid"},
		"Test-No Filename": {fileName: "",
			content:       testString,
			valid:         false,
			errorContains: "ends with a slash",
			errorMsg:      "TYPE: Invalid (no file)"},
		"Test-No Content": {fileName: fileName,
			content:       "",
			valid:         false,
			errorContains: "no bytes written",
			errorMsg:      "TYPE: Invalid (no content)"},
	}

	for redirectName, rc := range redirectCases {
		for filetypeName, ftc := range filetypeCases {
			name := fmt.Sprintf("%s-%s", redirectName, filetypeName)

			content, err := func() (string, error) {
				ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if rc.redirect && r.URL.Path == "/redirect" {
						http.Redirect(w, r, "/"+ftc.fileName, http.StatusFound)
					} else {
						if r.URL.String() == ("/" + ftc.fileName) {
							w.Write([]byte(ftc.content))
						}
					}
				}))
				defer ts.Close()

				cm := system.NewCleanupManager()
				ctx := context.Background()
				sp, err := NewStorage(cm)
				if err != nil {
					return "", fmt.Errorf("%s: failed to create storage provider", name)
				}

				serverURL := ts.URL
				finalURL := ""
				if rc.redirect {
					finalURL = ts.URL + "/redirect"
				} else {
					finalURL = serverURL + "/" + ftc.fileName
				}

				spec := model.StorageSpec{
					StorageSource: model.StorageSourceURLDownload,
					URL:           finalURL,
					Path:          "/inputs",
				}

				volume, err := sp.PrepareStorage(ctx, spec)
				if err != nil {
					return "", fmt.Errorf("%s: failed to prepare storage: %+v", name, err)
				}

				require.Equalf(s.T(), filepath.Join(spec.Path, ftc.fileName), volume.Target, "%s: expected valid to be %t", name, ftc.valid)

				file, err := os.Open(volume.Source)
				if err != nil {
					return "", fmt.Errorf("%s: failed to open file: %+v", name, err)
				}

				defer func() {
					if err = file.Close(); err != nil {
						require.Fail(s.T(), "failed to close file: %s", name)
					}
				}()

				content, err := ioutil.ReadAll(file)
				if err != nil {
					return "", fmt.Errorf("%s: failed to read file: %+v", name, err)
				}

				if len(content) == 0 {
					return "", fmt.Errorf("%s: file is empty", name)
				}

				return string(content), nil
			}()

			if ftc.valid {
				text := string(content)
				require.Equal(s.T(), ftc.content, text, "%s: content of file does not match", name)
			} else {
				require.Error(s.T(), err, "%s: expected error", name)
				require.Contains(s.T(),
					err.Error(),
					ftc.errorContains,
					"%s: error does not contain expected string",
					name)
			}
		}
	}
}

//go:build integration || !unit

package downloader_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/http"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/s3signed"
	"github.com/bacalhau-project/bacalhau/pkg/lib/gzip"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	s3test "github.com/bacalhau-project/bacalhau/pkg/s3/test"
	ipfssource "github.com/bacalhau-project/bacalhau/pkg/storage/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"

	ipfs2 "github.com/bacalhau-project/bacalhau/pkg/downloader/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
)

type DownloaderSuite struct {
	*s3test.HelperSuite
	cm               *system.CleanupManager
	ipfsClient       *ipfs.Client
	downloadSettings *downloader.DownloaderSettings
	downloadProvider downloader.DownloaderProvider
	s3Signer         *s3helper.ResultSigner
}

func (ds *DownloaderSuite) SetupSuite() {
	logger.ConfigureTestLogging(ds.T())

	endpoint := testutils.MustHaveIPFS(ds.T())
	var err error
	ds.ipfsClient, err = ipfs.NewClient(context.Background(), endpoint)
	require.NoError(ds.T(), err)

	ds.HelperSuite.SetupSuite()
	ds.s3Signer = s3helper.NewResultSigner(s3helper.ResultSignerParams{
		ClientProvider: ds.ClientProvider,
		Expiration:     5 * time.Minute,
	})

	ds.downloadProvider = provider.NewMappedProvider(
		map[string]downloader.Downloader{
			models.StorageSourceIPFS: ipfs2.NewIPFSDownloader(ds.ipfsClient),
			models.StorageSourceS3PreSigned: s3signed.NewDownloader(s3signed.DownloaderParams{
				HTTPDownloader: http.NewHTTPDownloader(),
			}),
		},
	)
}

// Before each test, reset the download settings to the default.
func (ds *DownloaderSuite) SetupTest() {
	ds.downloadSettings = &downloader.DownloaderSettings{
		Timeout: downloader.DefaultDownloadTimeout,
	}
}

func (ds *DownloaderSuite) TearDownSuite() {
	ds.Destroy()
}

func TestDownloaderSuite(t *testing.T) {
	helperSuite := s3test.NewTestHelper(t, s3test.HelperSuiteParams{
		BasePrefix: "integration-tests-downloader",
	})
	suite.Run(t, &DownloaderSuite{HelperSuite: helperSuite})
}

type mockResult struct {
	stdout   string
	stderr   string
	exitCode string
	outputs  map[string]string
	path     string
}

func (ds *DownloaderSuite) mockOutput(outputNames ...string) mockResult {
	dir := ds.T().TempDir()

	res := &mockResult{
		stdout:   ds.mockFile(dir, downloader.DownloadFilenameStdout),
		stderr:   ds.mockFile(dir, downloader.DownloadFilenameStderr),
		exitCode: ds.mockFile(dir, downloader.DownloadFilenameExitCode),
		outputs:  make(map[string]string),
		path:     dir,
	}
	for _, name := range outputNames {
		res.outputs[name] = ds.mockFile(dir, "outputs", name)
	}
	return *res
}

// Generates a test file at the given path filled with random data, ensuring
// that any parent directories for the file are also present.
func (ds *DownloaderSuite) mockFile(path ...string) string {
	file := filepath.Join(path...)
	ds.Require().NoError(os.MkdirAll(filepath.Dir(file), os.ModePerm))

	content := uuid.NewString()
	ds.Require().NoError(os.WriteFile(file, []byte(content), 0644))
	return content
}

// Publish to IPFS
func publishToIPFS(ds *DownloaderSuite, dir string) *models.SpecConfig {
	if ds.ipfsClient == nil {
		ds.T().Skip("IPFS connect is not available")
	}

	cid, err := ds.ipfsClient.Put(ds.Ctx, dir)
	require.NoError(ds.T(), err)
	return &models.SpecConfig{
		Type: models.StorageSourceIPFS,
		Params: ipfssource.Source{
			CID: cid,
		}.ToMap(),
	}
}

// Publish to S3
func publishToS3(ds *DownloaderSuite, dir string) *models.SpecConfig {
	publisherSpec := ds.PreparePublisherSpec(true)
	storageSpec := ds.PublishResultSilently(publisherSpec, dir)
	ds.Require().NoError(ds.s3Signer.Transform(ds.Ctx, &storageSpec))
	return &storageSpec
}

func publishToS3Unsigned(ds *DownloaderSuite, dir string) *models.SpecConfig {
	publisherSpec := ds.PreparePublisherSpec(false)
	storageSpec := ds.PublishResultSilently(publisherSpec, dir)
	return &storageSpec
}

// Requires that a file exists when the path is traversed downwards from the
// output directory.
func requireFileExists(ds *DownloaderSuite, path ...string) string {
	testPath := filepath.Join(ds.downloadSettings.OutputDir, filepath.Join(path...))
	_, err := os.Stat(testPath)
	ds.Require().NoError(err, "File does not exist: %s", testPath)
	return testPath
}

// Requires that a file exists with the specified contents when the path is
// traversed downwards from the output directory.
func requireFile(ds *DownloaderSuite, expected string, path ...string) {
	testPath := requireFileExists(ds, path...)

	contents, err := os.ReadFile(testPath)
	require.NoError(ds.T(), err)
	require.Equal(ds.T(), expected, string(contents))
}

type testCase struct {
	publishFn  func(*DownloaderSuite, string) *models.SpecConfig
	rawMatcher func(ds *DownloaderSuite, result *models.SpecConfig, rawParentPath string) string
	shouldFail bool
}

var publishers = map[string]testCase{
	models.StorageSourceS3PreSigned: {
		publishFn: publishToS3,
		rawMatcher: func(ds *DownloaderSuite, result *models.SpecConfig, rawParentPath string) string {
			dirEntries, err := os.ReadDir(rawParentPath)
			ds.Require().NoError(err)

			for _, entry := range dirEntries {
				sanitizedFileName, err := http.SanitizeFileName(result.Params["PreSignedURL"].(string))
				require.NoError(ds.T(), err)
				if entry.Name() == sanitizedFileName {
					sourcePath := filepath.Join(rawParentPath, entry.Name())
					uncompressedName, err := gzip.DecompressInPlace(sourcePath)
					require.NoError(ds.T(), err)
					return filepath.Base(uncompressedName)
				}
			}
			require.Failf(ds.T(), "Could not find raw file", "Could not find raw file for %s", result.Params["PreSignedURL"])
			return ""
		},
	},
	models.StorageSourceIPFS: {
		publishFn: publishToIPFS,
		rawMatcher: func(ds *DownloaderSuite, result *models.SpecConfig, rawParentPath string) string {
			dirEntries, err := os.ReadDir(rawParentPath)
			ds.Require().NoError(err)

			for _, entry := range dirEntries {
				if entry.Name() == result.Params["CID"].(string) {
					return entry.Name()
				}
			}
			require.Failf(ds.T(), "Could not find raw file", "Could not find raw file for %s", result.Params["CID"])
			return ""
		},
	},
	models.StorageSourceS3: {
		publishFn:  publishToS3Unsigned,
		shouldFail: true,
	},
}

func (ds *DownloaderSuite) getPublishers() map[string]testCase {
	res := make(map[string]testCase)
	for name, publisher := range publishers {
		if name == models.StorageSourceIPFS && ds.ipfsClient == nil {
			ds.T().Log("Skipping IPFS test as IPFS Connect is not available")
			continue
		}
		res[name] = publisher
	}
	return res
}

func (ds *DownloaderSuite) TestNoExpectedResults() {
	err := downloader.DownloadResults(
		ds.Ctx,
		[]*models.SpecConfig{},
		ds.downloadProvider,
		ds.downloadSettings,
	)
	require.NoError(ds.T(), err)
}

func (ds *DownloaderSuite) download(results ...*models.SpecConfig) error {
	ds.downloadSettings.OutputDir = ds.T().TempDir()
	return downloader.DownloadResults(
		ds.Ctx,
		results,
		ds.downloadProvider,
		ds.downloadSettings,
	)
}

func (ds *DownloaderSuite) TestSingleOutput() {
	for name, publisher := range ds.getPublishers() {
		ds.T().Run("TestSingleOutput: "+name, func(t *testing.T) {
			res := ds.mockOutput("hello.txt")
			err := ds.download(publisher.publishFn(ds, res.path))
			if publisher.shouldFail {
				require.Error(t, err)
				return
			}
			require.NoError(ds.T(), err)

			requireFile(ds, res.stdout, "stdout")
			requireFile(ds, res.stderr, "stderr")
			requireFile(ds, res.exitCode, "exitCode")
			requireFile(ds, res.outputs["hello.txt"], "outputs", "hello.txt")
		})
	}
}

func (ds *DownloaderSuite) TestSingleRawOutput() {
	for name, publisher := range ds.getPublishers() {
		ds.T().Run("TestSingleRawOutput: "+name, func(t *testing.T) {
			ds.downloadSettings.Raw = true
			res := ds.mockOutput("hello.txt", "goodbye.txt")

			publishedResult := publisher.publishFn(ds, res.path)
			err := ds.download(publishedResult)
			if publisher.shouldFail {
				require.Error(t, err)
				return
			}
			require.NoError(ds.T(), err)

			rawParentPath := filepath.Join(ds.downloadSettings.OutputDir, downloader.DownloadRawFolderName)
			rawPath := publisher.rawMatcher(ds, publishedResult, rawParentPath)
			requireFile(ds, res.stdout, downloader.DownloadRawFolderName, rawPath, "stdout")
			requireFile(ds, res.stderr, downloader.DownloadRawFolderName, rawPath, "stderr")
			requireFile(ds, res.exitCode, downloader.DownloadRawFolderName, rawPath, "exitCode")
			requireFile(ds, res.outputs["goodbye.txt"], downloader.DownloadRawFolderName, rawPath, "outputs", "goodbye.txt")
			requireFile(ds, res.outputs["hello.txt"], downloader.DownloadRawFolderName, rawPath, "outputs", "hello.txt")
		})
	}
}

func (ds *DownloaderSuite) TestMultiRawOutput() {
	for name, publisher := range ds.getPublishers() {
		ds.T().Run("TestMultiRawOutput: "+name, func(t *testing.T) {
			ds.downloadSettings.Raw = true
			res := ds.mockOutput("hello.txt")
			res2 := ds.mockOutput("goodbye.txt")
			publishedResult1 := publisher.publishFn(ds, res.path)
			publishedResult2 := publisher.publishFn(ds, res2.path)
			err := ds.download(publishedResult1, publishedResult2)
			if publisher.shouldFail {
				require.Error(t, err)
				return
			}
			require.NoError(ds.T(), err)

			rawParentPath := filepath.Join(ds.downloadSettings.OutputDir, downloader.DownloadRawFolderName)
			rawPath1 := publisher.rawMatcher(ds, publishedResult1, rawParentPath)
			rawPath2 := publisher.rawMatcher(ds, publishedResult2, rawParentPath)

			requireFile(ds, res.stdout, downloader.DownloadRawFolderName, rawPath1, "stdout")
			requireFile(ds, res.stderr, downloader.DownloadRawFolderName, rawPath1, "stderr")
			requireFile(ds, res.exitCode, downloader.DownloadRawFolderName, rawPath1, "exitCode")
			requireFile(ds, res.outputs["hello.txt"], downloader.DownloadRawFolderName, rawPath1, "outputs", "hello.txt")

			requireFile(ds, res2.stdout, downloader.DownloadRawFolderName, rawPath2, "stdout")
			requireFile(ds, res2.stderr, downloader.DownloadRawFolderName, rawPath2, "stderr")
			requireFile(ds, res2.exitCode, downloader.DownloadRawFolderName, rawPath2, "exitCode")
			requireFile(ds, res2.outputs["goodbye.txt"], downloader.DownloadRawFolderName, rawPath2, "outputs", "goodbye.txt")
		})
	}
}

func (ds *DownloaderSuite) TestMultiMergedOutput() {
	for name, publisher := range ds.getPublishers() {
		ds.Run("TestMultiMergedOutput: "+name, func() {
			res := ds.mockOutput("hello.txt")
			res2 := ds.mockOutput("goodbye.txt")
			err := ds.download(
				publisher.publishFn(ds, res.path),
				publisher.publishFn(ds, res2.path),
			)
			if publisher.shouldFail {
				require.Error(ds.T(), err)
				return
			}
			require.NoError(ds.T(), err)
			requireFile(ds, res.outputs["hello.txt"], "outputs", "hello.txt")
			requireFile(ds, res2.outputs["goodbye.txt"], "outputs", "goodbye.txt")
		})
	}
}

func (ds *DownloaderSuite) TestMultiMergeConflictingOutput() {
	for name, publisher := range ds.getPublishers() {
		ds.Run("TestMultiMergeConflictingOutput: "+name, func() {
			res := ds.mockOutput("same_same.txt")
			res2 := ds.mockOutput("same_same.txt")
			err := ds.download(
				publisher.publishFn(ds, res.path),
				publisher.publishFn(ds, res2.path),
			)
			require.Error(ds.T(), err)
		})
	}
}

func (ds *DownloaderSuite) TestOutputWithNoStdFiles() {
	for name, publisher := range ds.getPublishers() {
		ds.Run("TestOutputWithNoStdFiles: "+name, func() {
			path := ds.T().TempDir()
			ds.mockFile(path, "outputs", "lonely.txt")
			err := ds.download(
				publisher.publishFn(ds, path),
			)
			if publisher.shouldFail {
				require.Error(ds.T(), err)
				return
			}
			require.NoError(ds.T(), err)
			requireFileExists(ds, "outputs", "lonely.txt")
		})
	}
}

func (ds *DownloaderSuite) TestCustomVolumeNames() {
	for name, publisher := range ds.getPublishers() {
		ds.Run("TestCustomVolumeNames: "+name, func() {
			path := ds.T().TempDir()
			ds.mockFile(path, "secrets", "private.pem")
			err := ds.download(
				publisher.publishFn(ds, path),
			)
			if publisher.shouldFail {
				require.Error(ds.T(), err)
				return
			}
			require.NoError(ds.T(), err)
			requireFileExists(ds, "secrets", "private.pem")
		})
	}
}

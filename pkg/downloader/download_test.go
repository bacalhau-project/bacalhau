//go:build unit || !integration

package downloader

import (
	"context"
	"crypto/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"

	ipfs2 "github.com/bacalhau-project/bacalhau/pkg/downloader/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestDownloaderSuite(t *testing.T) {
	suite.Run(t, new(DownloaderSuite))
}

type DownloaderSuite struct {
	suite.Suite
	cm               *system.CleanupManager
	client           ipfs.Client
	outputDir        string
	downloadSettings *model.DownloaderSettings
	downloadProvider DownloaderProvider
}

func (ds *DownloaderSuite) SetupSuite() {
	logger.ConfigureTestLogging(ds.T())
	system.InitConfigForTesting(ds.T())
}

// Before each test
func (ds *DownloaderSuite) SetupTest() {
	ds.cm = system.NewCleanupManager()
	ds.T().Cleanup(func() {
		ds.cm.Cleanup(context.Background())
	})

	ctx, cancel := context.WithCancel(context.Background())
	ds.T().Cleanup(cancel)

	node, err := ipfs.NewLocalNode(ctx, ds.cm, nil)
	require.NoError(ds.T(), err)

	ds.client = node.Client()

	swarm, err := node.SwarmAddresses()
	require.NoError(ds.T(), err)

	testOutputDir := ds.T().TempDir()
	ds.outputDir = testOutputDir

	ds.downloadSettings = &model.DownloaderSettings{
		Timeout:        model.DefaultIPFSTimeout,
		OutputDir:      testOutputDir,
		IPFSSwarmAddrs: strings.Join(swarm, ","),
	}

	ds.downloadProvider = provider.NewMappedProvider(
		map[string]Downloader{
			model.StorageSourceIPFS.String(): ipfs2.NewIPFSDownloader(ds.cm, ds.downloadSettings),
		},
	)
}

type mockResult struct {
	cid      string
	stdout   []byte
	stderr   []byte
	exitCode []byte
	outputs  map[string][]byte
}

// Generate a file with random data.
func generateFile(path string) ([]byte, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer closer.CloseWithLogOnError("file", file)

	b := make([]byte, 128)
	_, err = rand.Read(b)
	if err != nil {
		return nil, err
	}

	_, err = file.Write(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Wraps generation of a set of output files that represent the output from a
// specific result, and saves them to IPFS.
//
// The passed setup func will be called with a temporary directory. Within the
// setup func, the user should make a number of calls to `mockFile` to generate
// files within the directory. At the end, the entire directory is saved to
// IPFS.
func mockOutput(ds *DownloaderSuite, setup func(string)) string {
	testDir := ds.T().TempDir()

	setup(testDir)

	cid, err := ds.client.Put(context.Background(), testDir)
	require.NoError(ds.T(), err)

	return cid
}

func (ds *DownloaderSuite) easyMockOutput(outputNames ...string) mockResult {
	dir := ds.T().TempDir()

	res := &mockResult{
		stdout:   mockFile(ds, dir, model.DownloadFilenameStdout),
		stderr:   mockFile(ds, dir, model.DownloadFilenameStderr),
		exitCode: mockFile(ds, dir, model.DownloadFilenameExitCode),
		outputs:  make(map[string][]byte),
	}
	for _, name := range outputNames {
		res.outputs[name] = mockFile(ds, dir, "outputs", name)
	}

	cid, err := ds.client.Put(context.Background(), dir)
	ds.NoError(err)
	res.cid = cid
	return *res
}

// Generates a test file at the given path filled with random data, ensuring
// that any parent directories for the file are also present.
func mockFile(ds *DownloaderSuite, path ...string) []byte {
	filePath := filepath.Join(path...)
	err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	require.NoError(ds.T(), err)

	contents, err := generateFile(filePath)
	require.NoError(ds.T(), err)
	return contents
}

// Requires that a file exists when the path is traversed downwards from the
// output directory.
func requireFileExists(ds *DownloaderSuite, path ...string) string {
	testPath := filepath.Join(ds.outputDir, filepath.Join(path...))
	require.FileExistsf(ds.T(), testPath, "File %s not present", testPath)

	return testPath
}

// Requires that a file exists with the specified contents when the path is
// traversed downwards from the output directory.
func requireFile(ds *DownloaderSuite, expected []byte, path ...string) {
	testPath := requireFileExists(ds, path...)

	contents, err := os.ReadFile(testPath)
	require.NoError(ds.T(), err)
	require.Equal(ds.T(), expected, contents)
}

func (ds *DownloaderSuite) TestNoExpectedResults() {
	err := DownloadResults(
		context.Background(),
		[]model.PublishedResult{},
		ds.downloadProvider,
		ds.downloadSettings,
	)
	require.NoError(ds.T(), err)
}

func (ds *DownloaderSuite) TestSingleOutput() {
	res := ds.easyMockOutput("hello.txt", "goodbye.txt")

	err := DownloadResults(
		context.Background(),
		[]model.PublishedResult{
			{
				NodeID: "testnode",
				Data: model.StorageSpec{
					StorageSource: model.StorageSourceIPFS,
					Name:          "result-0",
					CID:           res.cid,
				},
			},
		},
		ds.downloadProvider,
		ds.downloadSettings,
	)
	require.NoError(ds.T(), err)

	requireFile(ds, res.stdout, "stdout")
	requireFile(ds, res.stderr, "stderr")
	requireFile(ds, res.exitCode, "exitCode")
	requireFile(ds, res.outputs["goodbye.txt"], "outputs", "goodbye.txt")
	requireFile(ds, res.outputs["hello.txt"], "outputs", "hello.txt")
}

func (ds *DownloaderSuite) TestSingleRawOutput() {
	res := ds.easyMockOutput("hello.txt", "goodbye.txt")

	settings := ds.downloadSettings
	settings.Raw = true
	err := DownloadResults(
		context.Background(),
		[]model.PublishedResult{
			{
				NodeID: "testnode",
				Data: model.StorageSpec{
					StorageSource: model.StorageSourceIPFS,
					Name:          "result-0",
					CID:           res.cid,
				},
			},
		},
		ds.downloadProvider,
		settings,
	)
	require.NoError(ds.T(), err)

	requireFile(ds, res.stdout, model.DownloadCIDsFolderName, res.cid, "stdout")
	requireFile(ds, res.stderr, model.DownloadCIDsFolderName, res.cid, "stderr")
	requireFile(ds, res.exitCode, model.DownloadCIDsFolderName, res.cid, "exitCode")
	requireFile(ds, res.outputs["goodbye.txt"], model.DownloadCIDsFolderName, res.cid, "outputs", "goodbye.txt")
	requireFile(ds, res.outputs["hello.txt"], model.DownloadCIDsFolderName, res.cid, "outputs", "hello.txt")
}

func (ds *DownloaderSuite) TestMultiRawOutput() {
	res := ds.easyMockOutput("hello.txt")
	res2 := ds.easyMockOutput("goodbye.txt")

	settings := ds.downloadSettings
	settings.Raw = true
	err := DownloadResults(
		context.Background(),
		[]model.PublishedResult{
			{
				NodeID: "testnode",
				Data: model.StorageSpec{
					StorageSource: model.StorageSourceIPFS,
					Name:          "result-1",
					CID:           res.cid,
				},
			},
			{
				NodeID: "testnode",
				Data: model.StorageSpec{
					StorageSource: model.StorageSourceIPFS,
					Name:          "result-2",
					CID:           res2.cid,
				},
			},
		},
		ds.downloadProvider,
		settings,
	)
	require.NoError(ds.T(), err)

	requireFile(ds, res.stdout, model.DownloadCIDsFolderName, res.cid, "stdout")
	requireFile(ds, res.stderr, model.DownloadCIDsFolderName, res.cid, "stderr")
	requireFile(ds, res.exitCode, model.DownloadCIDsFolderName, res.cid, "exitCode")
	requireFile(ds, res.outputs["hello.txt"], model.DownloadCIDsFolderName, res.cid, "outputs", "hello.txt")

	requireFile(ds, res2.stdout, model.DownloadCIDsFolderName, res2.cid, "stdout")
	requireFile(ds, res2.stderr, model.DownloadCIDsFolderName, res2.cid, "stderr")
	requireFile(ds, res2.exitCode, model.DownloadCIDsFolderName, res2.cid, "exitCode")
	requireFile(ds, res2.outputs["goodbye.txt"], model.DownloadCIDsFolderName, res2.cid, "outputs", "goodbye.txt")
}

func (ds *DownloaderSuite) TestMultiMergedOutput() {
	res := ds.easyMockOutput("hello.txt")
	res2 := ds.easyMockOutput("goodbye.txt")

	err := DownloadResults(
		context.Background(),
		[]model.PublishedResult{
			{
				NodeID: "testnode",
				Data: model.StorageSpec{
					StorageSource: model.StorageSourceIPFS,
					Name:          "result-1",
					CID:           res.cid,
				},
			},
			{
				NodeID: "testnode",
				Data: model.StorageSpec{
					StorageSource: model.StorageSourceIPFS,
					Name:          "result-2",
					CID:           res2.cid,
				},
			},
		},
		ds.downloadProvider,
		ds.downloadSettings,
	)
	require.NoError(ds.T(), err)
	requireFile(ds, res.outputs["hello.txt"], "outputs", "hello.txt")
	requireFile(ds, res2.outputs["goodbye.txt"], "outputs", "goodbye.txt")
}

func (ds *DownloaderSuite) TestMultiMergeConflictingOutput() {
	res := ds.easyMockOutput("same_same.txt")
	res2 := ds.easyMockOutput("same_same.txt")

	err := DownloadResults(
		context.Background(),
		[]model.PublishedResult{
			{
				NodeID: "testnode",
				Data: model.StorageSpec{
					StorageSource: model.StorageSourceIPFS,
					Name:          "result-1",
					CID:           res.cid,
				},
			},
			{
				NodeID: "testnode",
				Data: model.StorageSpec{
					StorageSource: model.StorageSourceIPFS,
					Name:          "result-2",
					CID:           res2.cid,
				},
			},
		},
		ds.downloadProvider,
		ds.downloadSettings,
	)
	require.Error(ds.T(), err)
}

func (ds *DownloaderSuite) TestOutputWithNoStdFiles() {
	cid := mockOutput(ds, func(dir string) {
		mockFile(ds, dir, "outputs", "lonely.txt")
	})

	err := DownloadResults(
		context.Background(),
		[]model.PublishedResult{
			{
				NodeID: "testnode",
				Data: model.StorageSpec{
					StorageSource: model.StorageSourceIPFS,
					Name:          "result-0",
					CID:           cid,
				},
			},
		},
		ds.downloadProvider,
		ds.downloadSettings,
	)
	require.NoError(ds.T(), err)

	requireFileExists(ds, "outputs", "lonely.txt")
}

func (ds *DownloaderSuite) TestCustomVolumeNames() {
	cid := mockOutput(ds, func(s string) {
		mockFile(ds, s, "secrets", "private.pem")
	})

	err := DownloadResults(
		context.Background(),
		[]model.PublishedResult{
			{
				NodeID: "testnode",
				Data: model.StorageSpec{
					StorageSource: model.StorageSourceIPFS,
					Name:          "result-0",
					CID:           cid,
				},
			},
		},
		ds.downloadProvider,
		ds.downloadSettings,
	)
	require.NoError(ds.T(), err)

	requireFileExists(ds, "secrets", "private.pem")
}

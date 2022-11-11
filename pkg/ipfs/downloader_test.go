//go:build !integration

package ipfs

import (
	"context"
	"crypto/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// a normal test function and pass our suite to suite.Run
func TestDownloaderSuite(t *testing.T) {
	suite.Run(t, new(DownloaderSuite))
}

// Define the s, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type DownloaderSuite struct {
	suite.Suite
	cm               system.CleanupManager
	client           *Client
	outputDir        string
	downloadSettings IPFSDownloadSettings
}

// Before each test
func (ds *DownloaderSuite) SetupTest() {
	ds.cm = *system.NewCleanupManager()
	logger.ConfigureTestLogging(ds.T())
	require.NoError(ds.T(), system.InitConfigForTesting())

	node, err := NewLocalNode(context.Background(), &ds.cm, nil)
	require.NoError(ds.T(), err)

	client, err := node.Client()
	require.NoError(ds.T(), err)
	ds.client = client

	swarm, err := node.SwarmAddresses()
	require.NoError(ds.T(), err)

	testOutputDir := ds.T().TempDir()
	ds.outputDir = testOutputDir

	ds.downloadSettings = IPFSDownloadSettings{
		TimeoutSecs:    int(DefaultIPFSTimeout.Seconds()),
		OutputDir:      testOutputDir,
		IPFSSwarmAddrs: strings.Join(swarm, ","),
	}
}

func (ds *DownloaderSuite) TearDownTest() {
	ds.cm.Cleanup()
}

// Generate a file with random data.
func generateFile(path string) ([]byte, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

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
// specific shard, and saves them to IPFS.
//
// The passed setup func will be called with a temporary directory. Within the
// setup func, the user should make a number of calls to `mockFile` to generate
// files within the directory. At the end, the entire directory is saved to
// IPFS.
func mockShardOutput(ds *DownloaderSuite, setup func(string)) string {
	testDir := ds.T().TempDir()

	setup(testDir)

	cid, err := ds.client.Put(context.Background(), testDir)
	require.NoError(ds.T(), err)

	return cid
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
	err := DownloadJob(
		context.Background(),
		&ds.cm,
		[]model.StorageSpec{},
		[]model.PublishedResult{},
		*NewIPFSDownloadSettings(),
	)
	require.NoError(ds.T(), err)
}

func (ds *DownloaderSuite) TestFullOutput() {
	var exitCode, stdout, stderr, hello, goodbye []byte
	cid := mockShardOutput(ds, func(dir string) {
		exitCode = mockFile(ds, dir, "exitCode")
		stdout = mockFile(ds, dir, DownloadFilenameStdout)
		stderr = mockFile(ds, dir, "stderr")
		hello = mockFile(ds, dir, "outputs", "hello.txt")
		goodbye = mockFile(ds, dir, "outputs", "goodbye.txt")
	})

	err := DownloadJob(
		context.Background(),
		&ds.cm,
		[]model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				Name:          "outputs",
				Path:          "/outputs",
			},
		},
		[]model.PublishedResult{
			{
				NodeID:     "testnode",
				ShardIndex: 0,
				Data: model.StorageSpec{
					StorageSource: model.StorageSourceIPFS,
					Name:          "shard-0",
					CID:           cid,
				},
			},
		},
		ds.downloadSettings,
	)
	require.NoError(ds.T(), err)

	requireFile(ds, stdout, DownloadVolumesFolderName, "stdout")
	requireFile(ds, stderr, DownloadVolumesFolderName, "stderr")
	requireFile(ds, exitCode, DownloadShardsFolderName, "0_node_testnode", "exitCode")
	requireFile(ds, stdout, DownloadShardsFolderName, "0_node_testnode", "stdout")
	requireFile(ds, stderr, DownloadShardsFolderName, "0_node_testnode", "stderr")
	requireFile(ds, goodbye, DownloadVolumesFolderName, "outputs", "goodbye.txt")
	requireFile(ds, hello, DownloadVolumesFolderName, "outputs", "hello.txt")
}

func (ds *DownloaderSuite) TestOutputWithNoStdFiles() {
	cid := mockShardOutput(ds, func(dir string) {
		mockFile(ds, dir, "outputs", "lonely.txt")
	})

	err := DownloadJob(
		context.Background(),
		&ds.cm,
		[]model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				Name:          "outputs",
				Path:          "/outputs",
			},
		},
		[]model.PublishedResult{
			{
				NodeID:     "testnode",
				ShardIndex: 0,
				Data: model.StorageSpec{
					StorageSource: model.StorageSourceIPFS,
					Name:          "shard-0",
					CID:           cid,
				},
			},
		},
		ds.downloadSettings,
	)
	require.NoError(ds.T(), err)

	requireFileExists(ds, DownloadVolumesFolderName, "outputs", "lonely.txt")
}

func (ds *DownloaderSuite) TestOutputFromMultipleShards() {
	var shard0stdout, shard1stdout []byte
	cid0 := mockShardOutput(ds, func(s string) {
		shard0stdout = mockFile(ds, s, DownloadFilenameStdout)
		mockFile(ds, s, "outputs", "data0.csv")
	})

	cid1 := mockShardOutput(ds, func(s string) {
		shard1stdout = mockFile(ds, s, DownloadFilenameStdout)
		mockFile(ds, s, "outputs", "data1.csv")
	})

	err := DownloadJob(
		context.Background(),
		&ds.cm,
		[]model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				Name:          "outputs",
				Path:          "/outputs",
			},
		},
		[]model.PublishedResult{
			{
				NodeID:     "testnode",
				ShardIndex: 0,
				Data: model.StorageSpec{
					StorageSource: model.StorageSourceIPFS,
					Name:          "shard-0",
					CID:           cid0,
				},
			},
			{
				NodeID:     "testnode",
				ShardIndex: 1,
				Data: model.StorageSpec{
					StorageSource: model.StorageSourceIPFS,
					Name:          "shard-1",
					CID:           cid1,
				},
			},
		},
		ds.downloadSettings,
	)
	require.NoError(ds.T(), err)

	fullStdout := append(shard0stdout, shard1stdout...)
	requireFile(ds, fullStdout, DownloadVolumesFolderName, DownloadFilenameStdout)
	requireFile(ds, shard0stdout, DownloadShardsFolderName, "0_node_testnode", "stdout")
	requireFile(ds, shard1stdout, DownloadShardsFolderName, "1_node_testnode", "stdout")
	requireFileExists(ds, DownloadVolumesFolderName, "outputs", "data0.csv")
	requireFileExists(ds, DownloadVolumesFolderName, "outputs", "data1.csv")
}

func (ds *DownloaderSuite) TestCustomVolumeNames() {
	cid := mockShardOutput(ds, func(s string) {
		mockFile(ds, s, "secrets", "private.pem")
	})

	err := DownloadJob(
		context.Background(),
		&ds.cm,
		[]model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				Name:          "secrets",
				Path:          "/sensitive",
				// TODO: Path is currently ignored but is set on Docker jobs?
			},
		},
		[]model.PublishedResult{
			{
				NodeID:     "testnode",
				ShardIndex: 0,
				Data: model.StorageSpec{
					StorageSource: model.StorageSourceIPFS,
					Name:          "shard-0",
					CID:           cid,
				},
			},
		},
		ds.downloadSettings,
	)
	require.NoError(ds.T(), err)

	requireFileExists(ds, DownloadVolumesFolderName, "secrets", "private.pem")
}

package ipfs

import (
	"context"
	"crypto/rand"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
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
func (suite *DownloaderSuite) SetupTest() {
	suite.cm = *system.NewCleanupManager()
	require.NoError(suite.T(), system.InitConfigForTesting())

	node, err := NewLocalNode(context.Background(), &suite.cm, nil)
	require.NoError(suite.T(), err)

	client, err := node.Client()
	require.NoError(suite.T(), err)
	suite.client = client

	swarm, err := node.SwarmAddresses()
	require.NoError(suite.T(), err)

	testOutputDir, err := ioutil.TempDir(os.TempDir(), "bacalhau-downloader-test-outputs-*")
	require.NoError(suite.T(), err)
	suite.outputDir = testOutputDir

	suite.downloadSettings = IPFSDownloadSettings{
		TimeoutSecs:    int(DefaultIPFSTimeout.Seconds()),
		OutputDir:      testOutputDir,
		IPFSSwarmAddrs: strings.Join(swarm, ","),
	}
}

func (suite *DownloaderSuite) TearDownTest() {
	suite.cm.Cleanup()
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
func mockShardOutput(suite *DownloaderSuite, setup func(string)) string {
	testDir, err := ioutil.TempDir(os.TempDir(), "bacalhau-downloader-test-inputs-*")
	require.NoError(suite.T(), err)

	setup(testDir)

	cid, err := suite.client.Put(context.Background(), testDir)
	require.NoError(suite.T(), err)

	return cid
}

// Generates a test file at the given path filled with random data, ensuring
// that any parent directories for the file are also present.
func mockFile(suite *DownloaderSuite, path ...string) []byte {
	filePath := filepath.Join(path...)
	err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	require.NoError(suite.T(), err)

	contents, err := generateFile(filePath)
	require.NoError(suite.T(), err)
	return contents
}

// Requires that a file exists when the path is traversed downwards from the
// output directory.
func requireFileExists(suite *DownloaderSuite, path ...string) string {
	testPath := filepath.Join(suite.outputDir, filepath.Join(path...))
	require.FileExistsf(suite.T(), testPath, "File %s not present", testPath)

	return testPath
}

// Requires that a file exists with the specified contents when the path is
// traversed downwards from the output directory.
func requireFile(suite *DownloaderSuite, expected []byte, path ...string) {
	testPath := requireFileExists(suite, path...)

	contents, err := os.ReadFile(testPath)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), expected, contents)
}

func (suite *DownloaderSuite) TestNoExpectedResults() {
	err := DownloadJob(
		context.Background(),
		&suite.cm,
		[]model.StorageSpec{},
		[]model.StorageSpec{},
		*NewIPFSDownloadSettings(),
	)
	require.NoError(suite.T(), err)
}

func (suite *DownloaderSuite) TestFullOutput() {
	var exitCode, stdout, stderr, hello, goodbye []byte
	cid := mockShardOutput(suite, func(dir string) {
		exitCode = mockFile(suite, dir, "exitCode")
		stdout = mockFile(suite, dir, "stdout")
		stderr = mockFile(suite, dir, "stderr")
		hello = mockFile(suite, dir, "outputs", "hello.txt")
		goodbye = mockFile(suite, dir, "outputs", "goodbye.txt")
	})

	err := DownloadJob(
		context.Background(),
		&suite.cm,
		[]model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				Name:          "outputs",
				Path:          "/outputs",
			},
		},
		[]model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				Name:          "shard-0",
				CID:           cid,
			},
		},
		suite.downloadSettings,
	)
	require.NoError(suite.T(), err)

	requireFile(suite, stdout, "stdout")
	requireFile(suite, stderr, "stderr")
	requireFile(suite, exitCode, "shards", "shard-0", "exitCode")
	requireFile(suite, stdout, "shards", "shard-0", "stdout")
	requireFile(suite, stderr, "shards", "shard-0", "stderr")
	requireFile(suite, goodbye, "volumes", "outputs", "goodbye.txt")
	requireFile(suite, hello, "volumes", "outputs", "hello.txt")
}

func (suite *DownloaderSuite) TestOutputWithNoStdFiles() {
	cid := mockShardOutput(suite, func(dir string) {
		mockFile(suite, dir, "outputs", "lonely.txt")
	})

	err := DownloadJob(
		context.Background(),
		&suite.cm,
		[]model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				Name:          "outputs",
				Path:          "/outputs",
			},
		},
		[]model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				Name:          "shard-0",
				CID:           cid,
			},
		},
		suite.downloadSettings,
	)
	require.NoError(suite.T(), err)

	requireFileExists(suite, "volumes", "outputs", "lonely.txt")
}

func (suite *DownloaderSuite) TestOutputFromMultipleShards() {
	var shard0stdout, shard1stdout []byte
	cid0 := mockShardOutput(suite, func(s string) {
		shard0stdout = mockFile(suite, s, "stdout")
		mockFile(suite, s, "outputs", "data0.csv")
	})

	cid1 := mockShardOutput(suite, func(s string) {
		shard1stdout = mockFile(suite, s, "stdout")
		mockFile(suite, s, "outputs", "data1.csv")
	})

	err := DownloadJob(
		context.Background(),
		&suite.cm,
		[]model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				Name:          "outputs",
				Path:          "/outputs",
			},
		},
		[]model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				Name:          "shard-0",
				CID:           cid0,
			},
			{
				StorageSource: model.StorageSourceIPFS,
				Name:          "shard-1",
				CID:           cid1,
			},
		},
		suite.downloadSettings,
	)
	require.NoError(suite.T(), err)

	fullStdout := append(shard0stdout, shard1stdout...)
	requireFile(suite, fullStdout, "stdout")
	requireFile(suite, shard0stdout, "shards", "shard-0", "stdout")
	requireFile(suite, shard1stdout, "shards", "shard-1", "stdout")
	requireFileExists(suite, "volumes", "outputs", "data0.csv")
	requireFileExists(suite, "volumes", "outputs", "data1.csv")
}

func (suite *DownloaderSuite) TestCustomVolumeNames() {
	cid := mockShardOutput(suite, func(s string) {
		mockFile(suite, s, "secrets", "private.pem")
	})

	err := DownloadJob(
		context.Background(),
		&suite.cm,
		[]model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				Name:          "secrets",
				Path:          "/sensitive",
				// TODO: Path is currently ignored but is set on Docker jobs?
			},
		},
		[]model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				Name:          "shard-0",
				CID:           cid,
			},
		},
		suite.downloadSettings,
	)
	require.NoError(suite.T(), err)

	requireFileExists(suite, "volumes", "secrets", "private.pem")
}

// a normal test function and pass our suite to suite.Run
func TestDownloaderSuite(t *testing.T) {
	suite.Run(t, new(DownloaderSuite))
}

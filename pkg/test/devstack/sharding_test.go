package devstack

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	apicopy "github.com/filecoin-project/bacalhau/pkg/storage/ipfs_apicopy"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ShardingSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestShardingSuite(t *testing.T) {
	suite.Run(t, new(ShardingSuite))
}

// Before all suite
func (suite *ShardingSuite) SetupAllSuite() {

}

// Before each test
func (suite *ShardingSuite) SetupTest() {
	system.InitConfigForTesting(suite.T())
}

func (suite *ShardingSuite) TearDownTest() {
}

func (suite *ShardingSuite) TearDownAllSuite() {

}

func prepareFolderWithFoldersAndFiles(folderCount, fileCount int) (string, error) {
	basePath, err := os.MkdirTemp("", "sharding-test")
	if err != nil {
		return "", err
	}
	for i := 0; i < folderCount; i++ {
		subfolderPath := fmt.Sprintf("%s/folder%d", basePath, i)
		err = os.Mkdir(subfolderPath, 0700)
		if err != nil {
			return "", err
		}
		for j := 0; j < fileCount; j++ {
			err = os.WriteFile(
				fmt.Sprintf("%s/%d.txt", subfolderPath, j),
				[]byte(fmt.Sprintf("hello %d %d", i, j)),
				0644,
			)
			if err != nil {
				return "", err
			}
		}
	}
	return basePath, nil
}

func prepareFolderWithFiles(fileCount int) (string, error) {
	basePath, err := os.MkdirTemp("", "sharding-test")
	if err != nil {
		return "", err
	}
	for i := 0; i < fileCount; i++ {
		err = os.WriteFile(
			fmt.Sprintf("%s/%d.txt", basePath, i),
			[]byte(fmt.Sprintf("hello %d", i)),
			0644,
		)
		if err != nil {
			return "", err
		}
	}
	return basePath, nil
}

func (suite *ShardingSuite) TestExplodeCid() {
	const nodeCount = 1
	const folderCount = 10
	const fileCount = 10
	ctx, span := newSpan("sharding_explodecid")
	defer span.End()
	system.InitConfigForTesting(suite.T())

	cm := system.NewCleanupManager()

	stack, err := devstack.NewDevStackIPFS(cm, nodeCount)
	require.NoError(suite.T(), err)

	node := stack.Nodes[0]

	// make 10 folders each with 10 files
	dirPath, err := prepareFolderWithFoldersAndFiles(folderCount, fileCount)
	require.NoError(suite.T(), err)

	directoryCid, err := stack.AddFileToNodes(nodeCount, dirPath)
	require.NoError(suite.T(), err)

	ipfsProvider, err := apicopy.NewStorageProvider(cm, node.IpfsClient.APIAddress())
	require.NoError(suite.T(), err)

	results, err := ipfsProvider.Explode(ctx, storage.StorageSpec{
		Path:   "/input",
		Engine: storage.StorageSourceIPFS,
		Cid:    directoryCid,
	})
	require.NoError(suite.T(), err)

	resultPaths := []string{}
	for _, result := range results {
		resultPaths = append(resultPaths, result.Path)
	}

	// the top level node is en empty path
	expectedFilePaths := []string{"/input"}
	for i := 0; i < folderCount; i++ {
		expectedFilePaths = append(expectedFilePaths, fmt.Sprintf("/input/folder%d", i))
		for j := 0; j < fileCount; j++ {
			expectedFilePaths = append(expectedFilePaths, fmt.Sprintf("/input/folder%d/%d.txt", i, j))
		}
	}

	require.Equal(
		suite.T(),
		strings.Join(expectedFilePaths, ","),
		strings.Join(resultPaths, ","),
		"the exploded file paths do not match the expected ones",
	)
}

func (suite *ShardingSuite) TestEndToEnd() {
	ulimitValue := 0

	if _, err := exec.LookPath("ulimit"); err == nil {
		// Test to see how many files can be open on this system...
		cmd := exec.Command("ulimit", "-n")
		err := cmd.Run()
		require.NoError(suite.T(), err)

		out, _ := cmd.CombinedOutput()
		ulimitValue, err = strconv.Atoi(string(out))
		require.NoError(suite.T(), err)
	} else {
		var rLimit syscall.Rlimit
		err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
		require.NoError(suite.T(), err)
		ulimitValue, err = strconv.Atoi(fmt.Sprint(rLimit.Cur))
		require.NoError(suite.T(), err)
	}

	if ulimitValue <= 512 {
		suite.T().Skip("Skipping sharding end to end test because the ulimit value is too low.")
	}

	const totalFiles = 100
	const batchSize = 10
	const batchCount = totalFiles / batchSize
	const nodeCount = 3
	ctx, span := newSpan("sharding_endtoend")
	defer span.End()

	stack, cm := SetupTest(
		suite.T(),
		nodeCount,
		0,
		computenode.NewDefaultComputeNodeConfig(),
	)
	defer TeardownTest(stack, cm)

	dirPath, err := prepareFolderWithFiles(totalFiles)
	require.NoError(suite.T(), err)

	directoryCid, err := stack.AddFileToNodes(nodeCount, dirPath)
	require.NoError(suite.T(), err)

	jobSpec := executor.JobSpec{
		Engine:   executor.EngineDocker,
		Verifier: verifier.VerifierNoop,
		Docker: executor.JobSpecDocker{
			Image: "ubuntu:latest",
			Entrypoint: []string{
				"bash", "-c",
				// loop over each input file and write the filename to an output file named the same
				// thing in the results folder
				`for f in /input/*; do export filename=$(echo $f | sed 's/\/input//'); echo "hello $f" && echo "hello $f" >> /output/$filename; done`,
			},
		},
		Inputs: []storage.StorageSpec{
			{
				Engine: storage.StorageSourceIPFS,
				Cid:    directoryCid,
				Path:   "/input",
			},
		},
		Outputs: []storage.StorageSpec{
			{
				Engine: storage.StorageSourceIPFS,
				Name:   "results",
				Path:   "/output",
			},
		},
		Sharding: executor.JobShardingConfig{
			GlobPattern: "/input/*",
			BatchSize:   batchSize,
		},
	}

	jobDeal := executor.JobDeal{
		Concurrency: nodeCount,
	}

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)
	submittedJob, err := apiClient.Submit(ctx, jobSpec, jobDeal, nil)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), batchCount, submittedJob.ExecutionPlan.TotalShards)

	resolver := apiClient.GetJobStateResolver()
	err = resolver.WaitUntilComplete(ctx, submittedJob.ID)
	require.NoError(suite.T(), err)

	jobState, err := apiClient.GetJobState(ctx, submittedJob.ID)
	require.NoError(suite.T(), err)

	// each node should have run 10 shards because we have 3 nodes
	// and concurrency is 3
	nodeIDs, err := stack.GetNodeIds()
	require.NoError(suite.T(), err)

	for _, nodeID := range nodeIDs {
		nodeState, ok := jobState.Nodes[nodeID]
		require.True(suite.T(), ok)
		require.Equal(suite.T(), batchCount, len(nodeState.Shards))
	}

	jobResults, err := apiClient.GetResults(ctx, submittedJob.ID)
	require.NoError(suite.T(), err)

	downloadFolder, err := ioutil.TempDir("", "bacalhau-shard-test")
	require.NoError(suite.T(), err)

	swarmAddresses, err := stack.Nodes[0].IpfsNode.SwarmAddresses()
	require.NoError(suite.T(), err)

	log.Info().Msgf("Downloading results to %s", downloadFolder)

	err = ipfs.DownloadJob(
		cm,
		submittedJob,
		jobResults,
		ipfs.IPFSDownloadSettings{
			TimeoutSecs:    10,
			OutputDir:      downloadFolder,
			IPFSSwarmAddrs: strings.Join(swarmAddresses, ","),
		},
	)
	require.NoError(suite.T(), err)

	// check that the merged stdout is correct
	expectedStdoutArray := []string{}
	expectedResultsFiles := []string{}
	for i := 0; i < totalFiles; i++ {
		expectedStdoutArray = append(expectedStdoutArray, fmt.Sprintf("hello /input/%d.txt", i))
		expectedResultsFiles = append(expectedResultsFiles, fmt.Sprintf("%d.txt", i))
	}

	sort.Strings(expectedStdoutArray)
	sort.Strings(expectedResultsFiles)

	require.FileExists(suite.T(), filepath.Join(downloadFolder, "stdout"))
	actualStdoutBytes, err := os.ReadFile(filepath.Join(downloadFolder, "stdout"))
	require.NoError(suite.T(), err)

	actualStdoutArray := strings.Split(string(actualStdoutBytes), "\n")
	sort.Strings(actualStdoutArray)

	require.Equal(suite.T(), "\n"+strings.Join(expectedStdoutArray, "\n"), strings.Join(actualStdoutArray, "\n"), "the merged stdout is not correct")

	// check that we have a "results" output volume with all the files inside
	require.DirExists(suite.T(), filepath.Join(downloadFolder, "volumes", "results"))
	files, err := ioutil.ReadDir(filepath.Join(downloadFolder, "volumes", "results"))
	require.NoError(suite.T(), err)

	actualResultsFiles := []string{}

	for _, foundFile := range files {
		actualResultsFiles = append(actualResultsFiles, foundFile.Name())
	}

	sort.Strings(actualResultsFiles)

	require.Equal(suite.T(), strings.Join(expectedResultsFiles, "\n"), strings.Join(actualResultsFiles, "\n"), "the merged list of files is not correct")
}

func (suite *ShardingSuite) TestNoShards() {
	const nodeCount = 1
	ctx, span := newSpan("sharding_noshards")
	defer span.End()

	stack, cm := SetupTest(
		suite.T(),
		nodeCount,
		0,
		computenode.NewDefaultComputeNodeConfig(),
	)
	defer TeardownTest(stack, cm)

	dirPath, err := prepareFolderWithFiles(0)
	require.NoError(suite.T(), err)

	directoryCid, err := stack.AddFileToNodes(nodeCount, dirPath)
	require.NoError(suite.T(), err)

	jobSpec := executor.JobSpec{
		Engine:   executor.EngineDocker,
		Verifier: verifier.VerifierNoop,
		Docker: executor.JobSpecDocker{
			Image: "ubuntu:latest",
			Entrypoint: []string{
				"bash", "-c",
				`echo "where did all the files go?"`,
			},
		},
		Inputs: []storage.StorageSpec{
			{
				Engine: storage.StorageSourceIPFS,
				Cid:    directoryCid,
				Path:   "/input",
			},
		},
		Outputs: []storage.StorageSpec{},
		Sharding: executor.JobShardingConfig{
			GlobPattern: "/input/*",
			BatchSize:   1,
		},
	}

	jobDeal := executor.JobDeal{
		Concurrency: nodeCount,
	}

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)
	_, err = apiClient.Submit(ctx, jobSpec, jobDeal, nil)
	require.Error(suite.T(), err)
	require.True(suite.T(), strings.Contains(err.Error(), "no sharding atoms found for glob pattern"))
}

func (suite *ShardingSuite) TestExplodeVideos() {
	const nodeCount = 1
	ctx, span := newSpan("sharding_video_files")
	defer span.End()

	videos := []string{
		"Bird flying over the lake.mp4",
		"Calm waves on a rocky sea gulf.mp4",
		"Prominent Late Gothic styled architecture.mp4",
	}

	stack, cm := SetupTest(
		suite.T(),
		nodeCount,
		0,
		computenode.NewDefaultComputeNodeConfig(),
	)
	defer TeardownTest(stack, cm)

	dirPath, err := os.MkdirTemp("", "sharding-test")
	require.NoError(suite.T(), err)
	for _, video := range videos {
		err = os.WriteFile(
			fmt.Sprintf("%s/%s", dirPath, video),
			[]byte(fmt.Sprintf("hello %s", video)),
			0644,
		)
		require.NoError(suite.T(), err)
	}

	directoryCid, err := stack.AddFileToNodes(nodeCount, dirPath)
	require.NoError(suite.T(), err)

	jobSpec := executor.JobSpec{
		Engine:   executor.EngineDocker,
		Verifier: verifier.VerifierNoop,
		Docker: executor.JobSpecDocker{
			Image: "ubuntu:latest",
			Entrypoint: []string{
				"bash", "-c",
				`ls -la /inputs`,
			},
		},
		Inputs: []storage.StorageSpec{
			{
				Engine: storage.StorageSourceIPFS,
				Cid:    directoryCid,
				Path:   "/inputs",
			},
		},
		Outputs: []storage.StorageSpec{},
		Sharding: executor.JobShardingConfig{
			BasePath:    "/inputs",
			GlobPattern: "*.mp4",
			BatchSize:   1,
		},
	}

	jobDeal := executor.JobDeal{
		Concurrency: nodeCount,
	}

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)
	_, err = apiClient.Submit(ctx, jobSpec, jobDeal, nil)
	require.NoError(suite.T(), err)
}

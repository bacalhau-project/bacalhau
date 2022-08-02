package devstack

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
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
		Verifier: verifier.VerifierIpfs,
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

	err = ipfs.DownloadJob(
		cm,
		submittedJob,
		jobResults,
		ipfs.DownloadSettings{
			TimeoutSecs:    10,
			OutputDir:      downloadFolder,
			IPFSSwarmAddrs: strings.Join(swarmAddresses, ","),
		},
	)
	require.NoError(suite.T(), err)

	fmt.Printf("downloadFolder --------------------------------------\n")
	spew.Dump(downloadFolder)
}

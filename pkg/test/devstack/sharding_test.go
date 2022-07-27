package devstack

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
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

func prepareFolderWithFiles(folderCount, fileCount int) (string, error) {
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
	dirPath, err := prepareFolderWithFiles(folderCount, fileCount)
	require.NoError(suite.T(), err)

	directoryCid, err := stack.AddFileToNodes(nodeCount, dirPath)
	require.NoError(suite.T(), err)

	ipfsProvider, err := apicopy.NewStorageProvider(cm, node.IpfsClient.APIAddress())
	require.NoError(suite.T(), err)

	results, err := ipfsProvider.Explode(ctx, storage.StorageSpec{
		Engine: storage.StorageSourceIPFS,
		Cid:    directoryCid,
	})
	require.NoError(suite.T(), err)

	resultPaths := []string{}
	for _, result := range results {
		resultPaths = append(resultPaths, result.Path)
	}

	// the top level node is en empty path
	expectedFilePaths := []string{"/"}
	for i := 0; i < folderCount; i++ {
		expectedFilePaths = append(expectedFilePaths, fmt.Sprintf("/folder%d", i))
		for j := 0; j < fileCount; j++ {
			expectedFilePaths = append(expectedFilePaths, fmt.Sprintf("/folder%d/%d.txt", i, j))
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

	const nodeCount = 1
	ctx, span := newSpan("sharding_endtoend")
	defer span.End()

	stack, cm := SetupTest(
		suite.T(),
		nodeCount,
		0,
		computenode.NewDefaultComputeNodeConfig(),
	)
	defer TeardownTest(stack, cm)

	nodeIDs, err := stack.GetNodeIds()
	require.NoError(suite.T(), err)

	dirPath, err := prepareFolderWithFiles(10, 10)
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
			GlobPattern: "/*",
			BatchSize:   10,
		},
	}

	jobDeal := executor.JobDeal{
		Concurrency: nodeCount,
	}

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)
	submittedJob, err := apiClient.Submit(ctx, jobSpec, jobDeal, nil)
	require.NoError(suite.T(), err)

	// wait for the job to complete across all nodes
	err = stack.WaitForJob(ctx, submittedJob.ID,
		devstack.WaitForJobThrowErrors([]executor.JobStateType{
			executor.JobStateCancelled,
			executor.JobStateError,
		}),
		devstack.WaitForJobAllHaveState(nodeIDs, executor.JobStateComplete),
	)
	require.NoError(suite.T(), err)

	// loadedJob, ok, err := apiClient.Get(ctx, submittedJob.ID)
	// require.True(suite.T(), ok)
	// require.NoError(suite.T(), err)

	// for nodeID, state := range loadedJob.State {
	// 	node, err := stack.GetNode(ctx, nodeID)
	// 	require.NoError(suite.T(), err)

	// 	outputDir, err := ioutil.TempDir("", "bacalhau-ipfs-devstack-test")
	// 	require.NoError(suite.T(), err)

	// 	outputPath := filepath.Join(outputDir, state.ResultsID)
	// 	err = node.IpfsClient.Get(ctx, state.ResultsID, outputPath)
	// 	require.NoError(suite.T(), err)
	// 	fmt.Printf("FOLDER --------------------------------------\n")
	// 	spew.Dump(outputPath)

	// }
}

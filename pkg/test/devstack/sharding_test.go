//go:build !(unit && (windows || darwin))

package devstack

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	apicopy "github.com/filecoin-project/bacalhau/pkg/storage/ipfs_apicopy"
	"github.com/filecoin-project/bacalhau/pkg/system"
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
func (suite *ShardingSuite) SetupSuite() {

}

// Before each test
func (suite *ShardingSuite) SetupTest() {
	err := system.InitConfigForTesting()
	require.NoError(suite.T(), err)
}

func (suite *ShardingSuite) TearDownTest() {
}

func (suite *ShardingSuite) TearDownSuite() {

}

func prepareFolderWithFoldersAndFiles(t *testing.T, folderCount, fileCount int) (string, error) {
	basePath := t.TempDir()
	for i := 0; i < folderCount; i++ {
		subfolderPath := fmt.Sprintf("%s/folder%d", basePath, i)
		err := os.Mkdir(subfolderPath, 0700)
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

func prepareFolderWithFiles(t *testing.T, fileCount int) (string, error) {
	basePath := t.TempDir()
	for i := 0; i < fileCount; i++ {
		err := os.WriteFile(
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
	ctx := context.Background()
	cm := system.NewCleanupManager()

	err := system.InitConfigForTesting()
	require.NoError(suite.T(), err)

	stack, err := devstack.NewDevStackIPFS(ctx, cm, nodeCount)
	require.NoError(suite.T(), err)

	t := system.GetTracer()
	ctx, rootSpan := system.NewRootSpan(ctx, t, "pkg/test/devstack/shardingtest/explodecid")
	defer rootSpan.End()
	cm.RegisterCallback(system.CleanupTraceProvider)

	node := stack.IPFSClients[0]

	// make 10 folders each with 10 files
	dirPath, err := prepareFolderWithFoldersAndFiles(suite.T(), folderCount, fileCount)
	require.NoError(suite.T(), err)

	directoryCid, err := devstack.AddFileToNodes(ctx, dirPath, stack.IPFSClients[:nodeCount]...)
	require.NoError(suite.T(), err)

	ipfsProvider, err := apicopy.NewStorage(cm, node.APIAddress())
	require.NoError(suite.T(), err)

	results, err := ipfsProvider.Explode(ctx, model.StorageSpec{
		Path:          "/input",
		StorageSource: model.StorageSourceIPFS,
		CID:           directoryCid,
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
	shouldRun, err := shouldRunShardingTest()
	require.NoError(suite.T(), err)

	if !shouldRun {
		suite.T().Skip("Skipping sharding end to end test because the ulimit value is too low.")
	}

	const totalFiles = 100
	const batchSize = 10
	const batchCount = totalFiles / batchSize
	const nodeCount = 3

	ctx := context.Background()

	stack, cm := SetupTest(
		ctx,
		suite.T(),

		nodeCount,
		0,
		false,
		computenode.NewDefaultComputeNodeConfig(),
	)

	t := system.GetTracer()
	ctx, rootSpan := system.NewRootSpan(ctx, t, "pkg/test/devstack/shardingtest/testendtoend")
	defer rootSpan.End()
	cm.RegisterCallback(system.CleanupTraceProvider)

	dirPath, err := prepareFolderWithFiles(suite.T(), totalFiles)
	require.NoError(suite.T(), err)

	directoryCid, err := devstack.AddFileToNodes(ctx, dirPath, devstack.ToIPFSClients(stack.Nodes[:nodeCount])...)
	require.NoError(suite.T(), err)

	j := &model.Job{}
	j.Spec = model.Spec{
		Engine:    model.EngineDocker,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherIpfs,
		Docker: model.JobSpecDocker{
			Image: "ubuntu:latest",
			Entrypoint: []string{
				"bash", "-c",
				// loop over each input file and write the filename to an output file named the same
				// thing in the results folder
				`for f in /input/*; do export filename=$(echo $f | sed 's/\/input//'); echo "hello $f" && echo "hello $f" >> /output/$filename; done`,
			},
		},
		Inputs: []model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				CID:           directoryCid,
				Path:          "/input",
			},
		},
		Outputs: []model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				Name:          "results",
				Path:          "/output",
			},
		},
		Sharding: model.JobShardingConfig{
			GlobPattern: "/input/*",
			BatchSize:   batchSize,
		},
	}

	j.Deal = model.Deal{
		Concurrency: nodeCount,
	}

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)
	submittedJob, err := apiClient.Submit(ctx, j, nil)
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
	require.True(suite.T(), len(jobResults) > 0, "there should be > 0 results")

	downloadFolder := suite.T().TempDir()

	swarmAddresses, err := stack.Nodes[0].IPFSClient.SwarmAddresses(ctx)
	require.NoError(suite.T(), err)

	log.Info().Msgf("Downloading results to %s", downloadFolder)

	err = ipfs.DownloadJob(
		ctx,
		cm,
		submittedJob.Spec.Outputs,
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
	ctx := context.Background()

	stack, cm := SetupTest(
		ctx,
		suite.T(),

		nodeCount,
		0,
		false,
		computenode.NewDefaultComputeNodeConfig(),
	)

	t := system.GetTracer()
	ctx, rootSpan := system.NewRootSpan(ctx, t, "pkg/test/devstack/shardingtest/testnoshards")
	defer rootSpan.End()
	cm.RegisterCallback(system.CleanupTraceProvider)

	dirPath, err := prepareFolderWithFiles(suite.T(), 0)
	require.NoError(suite.T(), err)

	directoryCid, err := devstack.AddFileToNodes(ctx, dirPath, devstack.ToIPFSClients(stack.Nodes[:nodeCount])...)
	require.NoError(suite.T(), err)

	j := &model.Job{}
	j.Spec = model.Spec{
		Engine:    model.EngineDocker,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherNoop,
		Docker: model.JobSpecDocker{
			Image: "ubuntu:latest",
			Entrypoint: []string{
				"bash", "-c",
				`echo "where did all the files go?"`,
			},
		},
		Inputs: []model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				CID:           directoryCid,
				Path:          "/input",
			},
		},
		Outputs: []model.StorageSpec{},
		Sharding: model.JobShardingConfig{
			GlobPattern: "/input/*",
			BatchSize:   1,
		},
	}

	j.Deal = model.Deal{
		Concurrency: nodeCount,
	}

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)
	_, err = apiClient.Submit(ctx, j, nil)
	require.Error(suite.T(), err)
	require.True(suite.T(), strings.Contains(err.Error(), "no sharding atoms found for glob pattern"))
}

// "publicapi: error unmarshaling error response: invalid character 'e' looking for beginning of value"

func (suite *ShardingSuite) TestExplodeVideos() {
	const nodeCount = 1
	ctx := context.Background()
	stack, cm := SetupTest(
		ctx,
		suite.T(),

		nodeCount,
		0,
		false,
		computenode.NewDefaultComputeNodeConfig(),
	)

	t := system.GetTracer()
	ctx, rootSpan := system.NewRootSpan(ctx, t, "pkg/devstack/shardingtest/testexplodevideos")
	defer rootSpan.End()
	cm.RegisterCallback(system.CleanupTraceProvider)

	videos := []string{
		"Bird flying over the lake.mp4",
		"Calm waves on a rocky sea gulf.mp4",
		"Prominent Late Gothic styled architecture.mp4",
	}

	dirPath := suite.T().TempDir()
	for _, video := range videos {
		err := os.WriteFile(
			filepath.Join(dirPath, video),
			[]byte(fmt.Sprintf("hello %s", video)),
			0644,
		)
		require.NoError(suite.T(), err)
	}

	directoryCid, err := devstack.AddFileToNodes(ctx, dirPath, devstack.ToIPFSClients(stack.Nodes[:nodeCount])...)
	require.NoError(suite.T(), err)

	j := &model.Job{}
	j.Spec = model.Spec{
		Engine:    model.EngineDocker,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherNoop,
		Docker: model.JobSpecDocker{
			Image: "ubuntu:latest",
			Entrypoint: []string{
				"bash", "-c",
				`ls -la /inputs`,
			},
		},
		Inputs: []model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				CID:           directoryCid,
				Path:          "/inputs",
			},
		},
		Outputs: []model.StorageSpec{},
		Sharding: model.JobShardingConfig{
			BasePath:    "/inputs",
			GlobPattern: "*.mp4",
			BatchSize:   1,
		},
	}

	j.Deal = model.Deal{
		Concurrency: nodeCount,
	}

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)
	_, err = apiClient.Submit(ctx, j, nil)
	require.NoError(suite.T(), err)
}

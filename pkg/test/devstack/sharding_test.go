//go:build integration

package devstack

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/docker"
	"github.com/filecoin-project/bacalhau/pkg/executor/noop"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/requester/publicapi"
	ipfs_storage "github.com/filecoin-project/bacalhau/pkg/storage/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ShardingSuite struct {
	scenario.ScenarioRunner
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestShardingSuite(t *testing.T) {
	suite.Run(t, new(ShardingSuite))
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

func (suite *ShardingSuite) TestExplodeCid() {
	const nodeCount = 1
	const folderCount = 10
	const fileCount = 10
	ctx := context.Background()
	cm := system.NewCleanupManager()

	err := system.InitConfigForTesting(suite.T())
	require.NoError(suite.T(), err)

	stack, err := devstack.NewDevStackIPFS(ctx, cm, nodeCount)
	require.NoError(suite.T(), err)

	node := stack.IPFSClients[0]

	// make 10 folders each with 10 files
	dirPath, err := prepareFolderWithFoldersAndFiles(suite.T(), folderCount, fileCount)
	require.NoError(suite.T(), err)

	directoryCid, err := ipfs.AddFileToNodes(ctx, dirPath, stack.IPFSClients[:nodeCount]...)
	require.NoError(suite.T(), err)

	ipfsProvider, err := ipfs_storage.NewStorage(cm, node)
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
	docker.MustHaveDocker(suite.T())

	const totalFiles = 100
	const batchSize = 10
	const batchCount = totalFiles / batchSize
	const nodeCount = 3

	var assertShardCounts job.CheckStatesFunction = func(js model.JobState) (bool, error) {
		return len(js.Shards) == batchCount, nil
	}

	// check that the merged stdout is correct
	checks := []scenario.CheckResults{}
	for i := 0; i < totalFiles; i++ {
		for j := 0; j < nodeCount; j++ {
			content := fmt.Sprintf("hello /input/%d.txt", i)
			filename := filepath.Join("results", fmt.Sprintf("%d.txt", i))
			checks = append(checks,
				scenario.FileEquals(filename, content+"\n"),
				scenario.FileContains(model.DownloadFilenameStdout, content, totalFiles*3+1),
			)
		}
	}

	testScenario := scenario.Scenario{
		Stack: &scenario.StackConfig{
			DevStackOptions: &devstack.DevStackOptions{NumberOfHybridNodes: nodeCount},
		},
		Inputs: scenario.StoredFile(
			prepareFolderWithFiles(suite.T(), totalFiles),
			"/input",
		),
		Outputs: []model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				Name:          "results",
				Path:          "/output",
			},
		},
		Spec: model.Spec{
			Engine:    model.EngineDocker,
			Verifier:  model.VerifierNoop,
			Publisher: model.PublisherIpfs,
			Docker: model.JobSpecDocker{
				Image: "ubuntu:latest",
				Entrypoint: []string{
					"bash", "-c",
					// loop over each input file and write the filename to an
					// output file named the same thing in the results folder
					`for f in /input/*; do export filename=$(echo $f | sed 's/\/input//');` +
						`echo "hello $f" && echo "hello $f" >> /output/$filename; done`,
				},
			},
			Sharding: model.JobShardingConfig{
				GlobPattern: "/input/*",
				BatchSize:   batchSize,
			},
		},
		Deal: model.Deal{Concurrency: 3},
		JobCheckers: []job.CheckStatesFunction{
			assertShardCounts,
			job.WaitExecutionsThrowErrors([]model.ExecutionStateType{
				model.ExecutionStateFailed,
			}),
			job.WaitForExecutionStates(map[model.ExecutionStateType]int{
				model.ExecutionStateCompleted: nodeCount * batchCount,
			}),
		},
		ResultsChecker: scenario.ManyChecks(checks...),
	}

	suite.RunScenario(testScenario)
}

func (suite *ShardingSuite) TestNoShards() {
	const nodeCount = 1
	ctx := context.Background()

	stack, _ := testutils.SetupTest(
		ctx,
		suite.T(),

		nodeCount,
		0,
		false,
		node.NewComputeConfigWithDefaults(),
		node.NewRequesterConfigWithDefaults(),
	)

	dirPath := prepareFolderWithFiles(suite.T(), 0)
	directoryCid, err := ipfs.AddFileToNodes(ctx, dirPath, devstack.ToIPFSClients(stack.Nodes[:nodeCount])...)
	require.NoError(suite.T(), err)

	j := &model.Job{
		APIVersion: model.APIVersionLatest().String(),
	}
	j.Spec = model.Spec{
		Engine:    model.EngineWasm,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherNoop,
		Wasm:      scenario.WasmHelloWorld.Spec.Wasm,
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

	j.Spec.Deal = model.Deal{
		Concurrency: nodeCount,
	}

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewRequesterAPIClient(apiUri)
	_, err = apiClient.Submit(ctx, j)
	require.Error(suite.T(), err)
	require.True(suite.T(), strings.Contains(err.Error(), "no sharding atoms found for glob pattern"))
}

func (suite *ShardingSuite) TestExplodeVideos() {
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

	testScenario := scenario.Scenario{
		Stack: &scenario.StackConfig{
			ExecutorConfig: noop.ExecutorConfig{},
		},
		Inputs:   scenario.StoredFile(dirPath, "/inputs"),
		Contexts: scenario.WasmHelloWorld.Contexts,
		Spec: model.Spec{
			Engine:    model.EngineNoop,
			Verifier:  model.VerifierNoop,
			Publisher: model.PublisherNoop,
			Sharding: model.JobShardingConfig{
				BasePath:    "/inputs",
				GlobPattern: "*.mp4",
				BatchSize:   1,
			},
		},
		JobCheckers: scenario.WaitUntilSuccessful(len(videos)),
	}

	suite.RunScenario(testScenario)
}

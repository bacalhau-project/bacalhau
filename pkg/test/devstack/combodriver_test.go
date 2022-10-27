//go:build !(unit && (windows || darwin))

package devstack

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ComboDriverSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestComboDriverSuite(t *testing.T) {
	suite.Run(t, new(ComboDriverSuite))
}

// Before all suite
func (suite *ComboDriverSuite) SetupSuite() {

}

// Before each test
func (suite *ComboDriverSuite) SetupTest() {
	err := system.InitConfigForTesting()
	require.NoError(suite.T(), err)
}

func (suite *ComboDriverSuite) TearDownTest() {
}

func (suite *ComboDriverSuite) TearDownSuite() {

}

// Test that the combo driver gives preference to the filecoin unsealed driver
// also that this does not affect normal jobs where the CID resides on the IPFS driver
func (suite *ComboDriverSuite) TestComboDriver() {
	exampleText := "hello world"

	runTest := func(
		unsealedMode bool,
	) {
		cm := system.NewCleanupManager()
		ctx := context.Background()
		defer cm.Cleanup()

		cid := "apples"
		basePath := suite.T().TempDir()
		filePath := filepath.Join(basePath, "file.txt")
		if unsealedMode {
			os.MkdirAll(filepath.Join(basePath, cid), os.ModePerm)
			filePath = filepath.Join(basePath, cid, "file.txt")
		}
		err := os.WriteFile(
			filePath,
			[]byte(fmt.Sprintf(exampleText)),
			0644,
		)
		require.NoError(suite.T(), err)

		unsealedPath := ""

		if unsealedMode {
			unsealedPath = fmt.Sprintf("%s/{{.CID}}", basePath)
		}

		options := devstack.DevStackOptions{
			NumberOfNodes:        1,
			PublicIPFSMode:       true,
			FilecoinUnsealedPath: unsealedPath,
		}

		stack, err := devstack.NewStandardDevStack(ctx, cm, options, computenode.NewDefaultComputeNodeConfig())
		require.NoError(suite.T(), err)

		if !unsealedMode {
			directoryCid, err := devstack.AddFileToNodes(ctx, basePath, stack.Nodes[0].IPFSClient)
			require.NoError(suite.T(), err)
			cid = directoryCid
		}

		j := &model.Job{}
		j.Spec = model.Spec{
			Engine:    model.EngineDocker,
			Verifier:  model.VerifierNoop,
			Publisher: model.PublisherIpfs,
			Docker: model.JobSpecDocker{
				Image: "ubuntu:latest",
				Entrypoint: []string{
					"bash", "-c",
					`cat /inputs/file.txt`,
				},
			},
			Inputs: []model.StorageSpec{
				{
					StorageSource: model.StorageSourceIPFS,
					CID:           cid,
					Path:          "/inputs",
				},
			},
			Outputs: []model.StorageSpec{},
		}

		j.Deal = model.Deal{
			Concurrency: 1,
		}

		apiUri := stack.Nodes[0].APIServer.GetURI()
		apiClient := publicapi.NewAPIClient(apiUri)
		submittedJob, err := apiClient.Submit(ctx, j, nil)
		require.NoError(suite.T(), err)

		resolver := apiClient.GetJobStateResolver()

		for {
			err = resolver.Wait(
				ctx,
				submittedJob.ID,
				1,
				job.WaitThrowErrors([]model.JobStateType{
					model.JobStateError,
				}),
				job.WaitForJobStates(map[model.JobStateType]int{
					model.JobStateCompleted: 1,
				}),
			)
			if err != nil && strings.Contains(err.Error(), "too many open files") {
				// Flaky test, try again
				time.Sleep(5 * time.Second)
				continue
			} else {
				break
			}
		}

		require.NoError(suite.T(), err)

		shards, err := resolver.GetShards(ctx, submittedJob.ID)
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), 1, len(shards), "there should be 1 shard")

		shard := shards[0]

		node, err := stack.GetNode(ctx, shard.NodeID)
		require.NoError(suite.T(), err)

		outputDir := suite.T().TempDir()
		require.NotEmpty(suite.T(), shard.PublishedResult.CID)

		outputPath := filepath.Join(outputDir, shard.PublishedResult.CID)
		err = node.IPFSClient.Get(ctx, shard.PublishedResult.CID, outputPath)
		require.NoError(suite.T(), err)

		dat, err := os.ReadFile(fmt.Sprintf("%s/stdout", outputPath))
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), exampleText, string(dat))
	}

	runTest(false)
	runTest(true)
}

package devstack

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/controller"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	executor_util "github.com/filecoin-project/bacalhau/pkg/executor/util"
	"github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	publisher_util "github.com/filecoin-project/bacalhau/pkg/publisher/util"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	verifier_util "github.com/filecoin-project/bacalhau/pkg/verifier/util"
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
func (suite *ComboDriverSuite) SetupAllSuite() {

}

// Before each test
func (suite *ComboDriverSuite) SetupTest() {
	system.InitConfigForTesting(suite.T())
}

func (suite *ComboDriverSuite) TearDownTest() {
}

func (suite *ComboDriverSuite) TearDownAllSuite() {

}

func (suite *ComboDriverSuite) TestComboDriver() {
	exampleText := "hello world"
	cm := system.NewCleanupManager()
	ctx := context.Background()
	defer cm.Cleanup()

	unsealedPath, err := os.MkdirTemp("", "combo-driver-test-unsealed")
	require.NoError(suite.T(), err)

	getStorageProviders := func(ipfsMultiAddress string, nodeIndex int) (map[storage.StorageSourceType]storage.StorageProvider, error) {
		return executor_util.NewStandardStorageProviders(cm, executor_util.StandardStorageProviderOptions{
			IPFSMultiaddress: ipfsMultiAddress,
		})
	}
	getExecutors := func(
		ipfsMultiAddress string,
		nodeIndex int,
		ctrl *controller.Controller,
	) (
		map[executor.EngineType]executor.Executor,
		error,
	) {
		ipfsParts := strings.Split(ipfsMultiAddress, "/")
		ipfsSuffix := ipfsParts[len(ipfsParts)-1]
		return executor_util.NewStandardExecutors(
			cm,
			executor_util.StandardExecutorOptions{
				DockerID: fmt.Sprintf("devstacknode%d-%s", nodeIndex, ipfsSuffix),
				Storage: executor_util.StandardStorageProviderOptions{
					IPFSMultiaddress:     ipfsMultiAddress,
					FilecoinUnsealedPath: fmt.Sprintf("%s/{{.Cid}}", unsealedPath),
				},
			},
		)
	}
	getVerifiers := func(
		ipfsMultiAddress string,
		nodeIndex int,
		ctrl *controller.Controller,
	) (
		map[verifier.VerifierType]verifier.Verifier,
		error,
	) {
		return verifier_util.NewNoopVerifiers(cm, ctrl.GetStateResolver())
	}
	getPublishers := func(
		ipfsMultiAddress string,
		nodeIndex int,
		ctrl *controller.Controller,
	) (
		map[publisher.PublisherType]publisher.Publisher,
		error,
	) {
		return publisher_util.NewIPFSPublishers(cm, ctrl.GetStateResolver(), ipfsMultiAddress)
	}
	stack, err := devstack.NewDevStack(
		cm,
		1,
		0,
		getStorageProviders,
		getExecutors,
		getVerifiers,
		getPublishers,
		computenode.NewDefaultComputeNodeConfig(),
		"",
		false,
	)
	require.NoError(suite.T(), err)

	dirPath, err := os.MkdirTemp("", "combo-driver-test")
	require.NoError(suite.T(), err)
	err = os.WriteFile(
		fmt.Sprintf("%s/file.txt", dirPath),
		[]byte(fmt.Sprintf(exampleText)),
		0644,
	)
	require.NoError(suite.T(), err)

	directoryCid, err := stack.AddFileToNodes(1, dirPath)
	require.NoError(suite.T(), err)

	jobSpec := executor.JobSpec{
		Engine:    executor.EngineDocker,
		Verifier:  verifier.VerifierNoop,
		Publisher: publisher.PublisherIpfs,
		Docker: executor.JobSpecDocker{
			Image: "ubuntu:latest",
			Entrypoint: []string{
				"bash", "-c",
				`cat /inputs/file.txt`,
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
	}

	jobDeal := executor.JobDeal{
		Concurrency: 1,
	}

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)
	submittedJob, err := apiClient.Submit(ctx, jobSpec, jobDeal, nil)
	require.NoError(suite.T(), err)

	resolver := apiClient.GetJobStateResolver()

	err = resolver.Wait(
		ctx,
		submittedJob.ID,
		1,
		job.WaitThrowErrors([]executor.JobStateType{
			executor.JobStateCancelled,
			executor.JobStateError,
		}),
		job.WaitForJobStates(map[executor.JobStateType]int{
			executor.JobStatePublished: 1,
		}),
	)
	require.NoError(suite.T(), err)

	shards, err := resolver.GetShards(ctx, submittedJob.ID)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 1, len(shards), "there should be 1 shard")

	shard := shards[0]

	node, err := stack.GetNode(ctx, shard.NodeID)
	require.NoError(suite.T(), err)

	outputDir, err := ioutil.TempDir("", "bacalhau-ipfs-devstack-test")
	require.NoError(suite.T(), err)
	require.NotEmpty(suite.T(), shard.PublishedResult.Cid)

	outputPath := filepath.Join(outputDir, shard.PublishedResult.Cid)
	err = node.IpfsClient.Get(ctx, shard.PublishedResult.Cid, outputPath)
	require.NoError(suite.T(), err)

	dat, err := os.ReadFile(fmt.Sprintf("%s/stdout", outputPath))
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), exampleText, string(dat))
}

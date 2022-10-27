//go:build !(unit && (windows || darwin))

package devstack

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type PublishOnErrorSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestPublishOnErrorSuite(t *testing.T) {
	suite.Run(t, new(PublishOnErrorSuite))
}

// Before all suite
func (s *PublishOnErrorSuite) SetupSuite() {

}

// Before each test
func (s *PublishOnErrorSuite) SetupTest() {
	err := system.InitConfigForTesting()
	require.NoError(s.T(), err)
}

func (suite *PublishOnErrorSuite) TearDownTest() {
}

func (s *PublishOnErrorSuite) TearDownSuite() {

}

func (s *PublishOnErrorSuite) TestPublishOnError() {
	stdoutText := "I am a miserable failure"
	ctx := context.Background()

	stack, cm := SetupTest(
		ctx,
		s.T(),
		1,
		0,
		false,
		computenode.NewDefaultComputeNodeConfig(),
	)

	t := system.GetTracer()
	ctx, rootSpan := system.NewRootSpan(ctx, t, "pkg/test/devstack/publish_on_error/test")
	defer rootSpan.End()
	cm.RegisterCallback(system.CleanupTraceProvider)

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)

	j := &model.Job{}
	j.Spec = model.Spec{
		Engine:    model.EngineDocker,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherIpfs,
		Docker: model.JobSpecDocker{
			Image: "ubuntu",
			Entrypoint: []string{
				"bash",
				"-c",
				fmt.Sprintf("echo %s && exit 1", stdoutText),
			},
		},
	}
	j.Deal = model.Deal{Concurrency: 1}

	submittedJob, err := apiClient.Submit(ctx, j, nil)
	require.NoError(s.T(), err)

	resolver := apiClient.GetJobStateResolver()

	err = resolver.Wait(
		ctx,
		submittedJob.ID,
		1,
		job.WaitForJobStates(map[model.JobStateType]int{
			model.JobStateCompleted: 1,
		}),
	)
	require.NoError(s.T(), err)

	shards, err := resolver.GetShards(ctx, submittedJob.ID)
	require.NoError(s.T(), err)

	shard := shards[0]

	node, err := stack.GetNode(ctx, shard.NodeID)
	require.NoError(s.T(), err)

	outputDir := s.T().TempDir()
	require.NotEmpty(s.T(), shard.PublishedResult.CID)

	outputPath := filepath.Join(outputDir, shard.PublishedResult.CID)
	err = node.IPFSClient.Get(ctx, shard.PublishedResult.CID, outputPath)
	require.NoError(s.T(), err)

	stdout, err := os.ReadFile(fmt.Sprintf("%s/stdout", outputPath))
	require.NoError(s.T(), err)

	require.Equal(s.T(), fmt.Sprintf("%s\n", stdoutText), string(stdout))
}

//go:build integration || !unit

package orchestrator_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"encoding/base64"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/eventhandler"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/transformer"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/translation"
	"github.com/stretchr/testify/suite"
	gomock "go.uber.org/mock/gomock"
)

type EndpointSuite struct {
	suite.Suite

	ctx context.Context
	cm  *system.CleanupManager

	client ipfs.Client
	node   *ipfs.Node
}

func TestEndpointSuite(t *testing.T) {
	suite.Run(t, new(EndpointSuite))
}

func (s *EndpointSuite) SetupSuite() {
	s.ctx = context.Background()
	s.cm = system.NewCleanupManager()

	node, _ := ipfs.NewNodeWithConfig(s.ctx, s.cm, types.IpfsConfig{PrivateInternal: true})
	s.node = node

	s.client = ipfs.NewClient(s.node.Client().API)
}

func (s *EndpointSuite) TearDownSuite() {
	s.node.Close(s.ctx)
}

func (s *EndpointSuite) TestInlinePinnerTransformInSubmit() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	tracerContextProvider := eventhandler.NewTracerContextProvider("test")
	localJobEventConsumer := eventhandler.NewChainedJobEventHandler(tracerContextProvider)
	eventEmitter := orchestrator.NewEventEmitter(orchestrator.EventEmitterParams{
		EventConsumer: localJobEventConsumer,
	})

	storageFactory := node.NewStandardStorageProvidersFactory()
	storageProviders, err := storageFactory.Get(s.ctx, node.NodeConfig{
		CleanupManager: s.cm,
		IPFSClient:     s.client,
	})
	s.Require().NoError(err)

	evalBroker := orchestrator.NewMockEvaluationBroker(ctrl)
	evalBroker.EXPECT().Enqueue(gomock.Any()).Return(nil)

	endpoint := orchestrator.NewBaseEndpoint(&orchestrator.BaseEndpointParams{
		ID:               "test_endpoint",
		EvaluationBroker: evalBroker,
		Store:            inmemory.NewInMemoryJobStore(),
		EventEmitter:     eventEmitter,
		JobTransformer: transformer.ChainedTransformer[*models.Job]{
			transformer.JobFn(transformer.IDGenerator),
			transformer.NewInlineStoragePinner(storageProviders),
		},
		TaskTranslator: translation.NewStandardTranslators(),
	})

	sb := strings.Builder{}
	sb.Grow(1024 * 10)
	for i := 0; i < 1024; i++ {
		_, _ = sb.WriteString("HelloWorld")
	}
	base64Content := base64.StdEncoding.EncodeToString([]byte(sb.String()))

	request := &orchestrator.SubmitJobRequest{
		&models.Job{
			Name: "testjob",
			Type: "batch",
			Tasks: []*models.Task{
				{
					Name:   "Task 1",
					Engine: &models.SpecConfig{Type: models.EngineNoop},
					InputSources: []*models.InputSource{
						{
							Source: &models.SpecConfig{
								Type: models.StorageSourceInline,
								Params: map[string]interface{}{
									"URL": fmt.Sprintf("data:text/html;base64,%s", base64Content),
								},
							},
							Target: "fake-target",
						},
					},
				},
			},
		},
	}

	response, err := endpoint.SubmitJob(s.ctx, request)
	s.Require().NoError(err)
	s.Require().NotEmpty(response.JobID)
	s.Require().NotEmpty(response.EvaluationID)
	s.Require().Empty(response.Warnings)

	// Because we are using pointers and calling directly we expect the job in the request
	// to have been transformed, with the input source now being IPFS.
	isource := request.Job.Task().InputSources[0]
	s.Require().Equal(isource.Source.Type, "ipfs")
	s.Require().NotEmpty(isource.Source.Params["CID"])

	cid := isource.Source.Params["CID"].(string)

	// We can't rely on GetCidSize to retrieve an accurate file size, it seems to
	// be out by 11 bytes, so we'll actually fetch the content
	// size, err := s.client.GetCidSize(s.ctx, cid)
	// s.Require().NoError(err)
	// s.Require().Equal(sb.Len(), size) // 10240 == 10251?

	tmpDir := s.T().TempDir()
	target := filepath.Join(tmpDir, "inline-transform-test.txt")

	err = s.client.Get(s.ctx, cid, target)
	s.Require().NoError(err)

	fileinfo, err := os.Stat(target)
	s.Require().NoError(err)
	s.Require().Equal(int64(sb.Len()), fileinfo.Size())

}

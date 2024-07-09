//go:build integration || !unit

package compute

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	dockermodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	"github.com/bacalhau-project/bacalhau/pkg/test/teststack"
	nodeutils "github.com/bacalhau-project/bacalhau/pkg/test/utils/node"
	"github.com/stretchr/testify/suite"
)

type CancelJobTestSuite struct {
	suite.Suite
}

func TestCancelJobTestSuite(t *testing.T) {
	suite.Run(t, new(CancelJobTestSuite))
}

func (s *CancelJobTestSuite) TestJobCancellation() {
	ctx := context.Background()

	docker.MustHaveDocker(s.T())

	fsr, c := setup.SetupBacalhauRepoForTesting(s.T())

	computeConfig, err := node.NewComputeConfigWith(c.Node.ComputeStoragePath, node.ComputeConfigParams{
		TotalResourceLimits: models.Resources{
			CPU:    1,
			Memory: 1 * 1024 * 1024 * 1024,
			Disk:   1 * 1024 * 1024 * 1024,
		},
		IgnorePhysicalResourceLimits: true,
		ControlPlaneSettings: types.ComputeControlPlaneConfig{
			ResourceUpdateFrequency: types.Duration(50 * time.Millisecond),
		},
	})

	s.Require().NoError(err)

	requesterConfig, err := node.NewRequesterConfigWith(node.RequesterConfigParams{
		NodeOverSubscriptionFactor: 2,
	})
	s.Require().NoError(err)

	stack := teststack.Setup(ctx, s.T(), fsr, c,
		devstack.WithNumberOfRequesterOnlyNodes(1),
		devstack.WithNumberOfComputeOnlyNodes(1),
		devstack.WithComputeConfig(computeConfig),
		devstack.WithRequesterConfig(requesterConfig),
	)

	nodeutils.WaitForNodeDiscovery(s.T(), stack.Nodes[0].RequesterNode, 2)

	es, err := dockermodels.NewDockerEngineBuilder("ubuntu").
		WithEntrypoint("bash", "-c", "sleep 100000").
		Build()

	s.Require().NoError(err)
	task := mock.TaskBuilder().
		Engine(es).
		BuildOrDie()
	job := mock.Job()
	job.Tasks[0] = task

	job.Normalize()

	apiServer := stack.Nodes[0].APIServer
	apiProtocol := "http"
	apiHost := apiServer.Address
	apiPort := apiServer.Port
	api := clientv2.New(fmt.Sprintf("%s://%s:%d", apiProtocol, apiHost, apiPort))

	submittedJob, err := api.Jobs().Put(ctx, &apimodels.PutJobRequest{
		Job: job,
	})
	s.Require().NoError(err)

	resolver := scenario.NewStateResolverFromAPI(api)
	time.Sleep(time.Second * 10)
	resolver.Wait(ctx, submittedJob.JobID, func(s *scenario.JobState) (bool, error) {
		if s.State.StateType == models.JobStateTypeRunning {
			return true, nil
		} else {
			return false, nil
		}
	})

	_, err = api.Jobs().Stop(ctx, &apimodels.StopJobRequest{
		JobID:  submittedJob.JobID,
		Reason: "Stop the job",
	})
	s.Require().NoError(err)

	resolver.Wait(ctx, submittedJob.JobID, func(js *scenario.JobState) (bool, error) {
		s.Require().Len(js.Executions, 1)
		if js.Executions[0].ComputeState.StateType == models.ExecutionStateCancelled {
			return true, nil
		} else {
			return false, nil
		}
	})

}

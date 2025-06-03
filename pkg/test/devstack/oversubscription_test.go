//go:build integration || !unit

package devstack

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/test/teststack"
	nodeutils "github.com/bacalhau-project/bacalhau/pkg/test/utils/node"
)

// OverSubscriptionTestSuite tests node over subscription behaviour where
// orchestrator node can be configured to allocate more jobs to compute nodes that
// exceed their availability capacity and trigger local queueing of jobs
type OverSubscriptionTestSuite struct {
	suite.Suite
	requester *node.Node
	compute   *node.Node
	client    clientv2.API

	jobResources            models.ResourcesConfig
	jobRunDuration          time.Duration
	jobRunWait              time.Duration
	resourceUpdateFrequency time.Duration
}

func (s *OverSubscriptionTestSuite) SetupSuite() {
	s.jobRunDuration = 2000 * time.Millisecond
	s.jobRunWait = 1000 * time.Millisecond
	s.resourceUpdateFrequency = 200 * time.Millisecond
	s.jobResources = models.ResourcesConfig{
		CPU:    "1",
		Memory: "1",
		Disk:   "1",
	}
}

func (s *OverSubscriptionTestSuite) setupStack(overSubscriptionFactor float64) {
	logger.ConfigureTestLogging(s.T())
	ctx := context.Background()

	testConfig, err := config.NewTestConfig()
	s.Require().NoError(err)

	stack := teststack.Setup(ctx, s.T(),
		devstack.WithNumberOfRequesterOnlyNodes(1),
		devstack.WithNumberOfComputeOnlyNodes(1),
		devstack.WithSystemConfig(node.SystemConfig{
			OverSubscriptionFactor: overSubscriptionFactor,
		}),
		devstack.WithBacalhauConfigOverride(types.Bacalhau{
			Compute: types.Compute{
				AllocatedCapacity: types.ResourceScalerFromModelsResourceConfig(s.jobResources),
				Heartbeat: types.Heartbeat{
					Interval: types.Duration(s.resourceUpdateFrequency),
				},
			},
			JobDefaults: types.JobDefaults{
				Batch: types.BatchJobDefaultsConfig{
					Task: types.BatchTaskDefaultConfig{
						Resources: types.FromModelsResourceConfig(s.jobResources),
					},
				},
			},
		}),
		teststack.WithNoopExecutor(
			noop_executor.ExecutorConfig{
				ExternalHooks: noop_executor.ExecutorConfigExternalHooks{
					JobHandler: noop_executor.DelayedJobHandler(s.jobRunDuration),
				},
			}, testConfig.Engines),
	)

	s.requester = stack.Nodes[0]
	s.compute = stack.Nodes[1]
	s.client = clientv2.New(s.requester.APIServer.GetURI().String())
	nodeutils.WaitForNodeDiscovery(s.T(), s.requester.RequesterNode, 2)
}

func (s *OverSubscriptionTestSuite) TearDownSuite() {
	if s.requester != nil {
		s.requester.CleanupManager.Cleanup(context.Background())
	}
	if s.compute != nil {
		s.compute.CleanupManager.Cleanup(context.Background())
	}
}

func TestOverSubscriptionTestSuite(t *testing.T) {
	suite.Run(t, new(OverSubscriptionTestSuite))
}

func (s *OverSubscriptionTestSuite) TestOverSubscribeNode() {
	tests := []struct {
		overSubscriptionFactor float64
		expectedStates         []models.JobStateType
		description            string
	}{
		{
			description:            "No over-subscription, one job scheduled",
			overSubscriptionFactor: 1,
			expectedStates: []models.JobStateType{
				models.JobStateTypeCompleted, models.JobStateTypeFailed, models.JobStateTypeFailed},
		},
		{
			description:            "2x over-subscription, two jobs scheduled",
			overSubscriptionFactor: 2,
			expectedStates: []models.JobStateType{
				models.JobStateTypeCompleted, models.JobStateTypeCompleted, models.JobStateTypeFailed},
		},
		{
			description:            "3x over-subscription, three jobs scheduled",
			overSubscriptionFactor: 3,
			expectedStates: []models.JobStateType{
				models.JobStateTypeCompleted, models.JobStateTypeCompleted, models.JobStateTypeCompleted},
		},
	}

	for _, tt := range tests {
		s.Run(tt.description, func() {
			s.setupStack(tt.overSubscriptionFactor)
			defer s.TearDownSuite()

			ctx := context.Background()
			jobsCount := len(tt.expectedStates)

			jobs := make([]*models.Job, jobsCount)
			for i := 0; i < jobsCount; i++ {
				job := mock.Job()
				job.Name = fmt.Sprintf("%s-%s", job.Name, job.ID)
				job.Task().ResourcesConfig = &s.jobResources
				jobs[i] = job
			}

			jobIDs := make([]string, jobsCount)
			for i, job := range jobs {
				submittedJob, err := s.client.Jobs().Put(ctx, &apimodels.PutJobRequest{
					Job: job,
				})
				s.NoError(err)
				jobIDs[i] = submittedJob.JobID
				time.Sleep(s.jobRunWait) // wait for requester to hear about the node resource updates
			}

			s.Eventuallyf(func() bool {
				jobStates := s.getJobStates()
				if len(jobStates) != len(tt.expectedStates) {
					return false
				}
				for i := range jobStates {
					if tt.expectedStates[i] != jobStates[i] {
						return false
					}
				}
				return true

			}, s.jobRunDuration*time.Duration(jobsCount)+time.Second, 500*time.Millisecond,
				"expected states: %s, retrieved states: %s", tt.expectedStates, s.getJobStates())
		})
	}
}

func (s *OverSubscriptionTestSuite) getJobStates() []models.JobStateType {
	jobStates := make([]models.JobStateType, 0)
	res, err := s.client.Jobs().List(context.Background(), &apimodels.ListJobsRequest{})
	s.Require().NoError(err)
	for _, j := range res.Items {
		jobStates = append(jobStates, j.State.StateType)
	}
	return jobStates
}

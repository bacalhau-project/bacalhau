//go:build integration || !unit

package requester

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
	"golang.org/x/exp/slices"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	"github.com/bacalhau-project/bacalhau/pkg/test/teststack"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	noop_publisher "github.com/bacalhau-project/bacalhau/pkg/publisher/noop"
	nodeutils "github.com/bacalhau-project/bacalhau/pkg/test/utils/node"
)

var errExecution = errors.New("i am a bad executor")
var errPublish = errors.New("i am a bad publisher")
var slowExecutorSleep = 5 * time.Second
var goodExecutorSleep = 250 * time.Millisecond // force bad executors to finish first to have more predictable tests

type RetriesSuite struct {
	suite.Suite
	requester     *node.Node
	computeNodes  []*node.Node
	client        clientv2.API
	stateResolver *scenario.StateResolver
}

func (s *RetriesSuite) SetupSuite() {
	logger.ConfigureTestLogging(s.T())
	fsr, cfg := setup.SetupBacalhauRepoForTesting(s.T())

	executionDir, err := cfg.ExecutionDir()
	s.Require().NoError(err)
	computeConfig, err := node.NewComputeConfigWith(executionDir, node.ComputeConfigParams{
		BidSemanticStrategy: bidstrategy.NewFixedBidStrategy(false, false),
		BidResourceStrategy: bidstrategy.NewFixedBidStrategy(false, false),
	})
	s.Require().NoError(err)
	nodeOverrides := []node.NodeConfig{
		{
			Labels: map[string]string{
				"name": "requester-node",
			},
			DependencyInjector: node.NodeDependencyInjector{},
		},
		{
			Labels: map[string]string{
				"name": "bid-rejector",
			},
			ComputeConfig: computeConfig,
		},
		{
			Labels: map[string]string{
				"name": "bad-executor",
			},
			DependencyInjector: node.NodeDependencyInjector{
				ExecutorsFactory: devstack.NewNoopExecutorsFactoryWithConfig(noop_executor.ExecutorConfig{
					ExternalHooks: noop_executor.ExecutorConfigExternalHooks{
						JobHandler: noop_executor.ErrorJobHandler(errExecution),
					},
				}),
			},
		},
		{
			Labels: map[string]string{
				"name": "bad-publisher",
			},
			DependencyInjector: node.NodeDependencyInjector{
				PublishersFactory: devstack.NewNoopPublishersFactoryWithConfig(noop_publisher.PublisherConfig{
					ExternalHooks: noop_publisher.PublisherExternalHooks{
						PublishResult: noop_publisher.ErrorResultPublisher(errPublish),
					},
				}),
			},
		},
		{
			Labels: map[string]string{
				"name": "slow-executor",
			},
			DependencyInjector: node.NodeDependencyInjector{
				ExecutorsFactory: devstack.NewNoopExecutorsFactoryWithConfig(noop_executor.ExecutorConfig{
					ExternalHooks: noop_executor.ExecutorConfigExternalHooks{
						JobHandler: noop_executor.DelayedJobHandler(slowExecutorSleep),
					},
				}),
			},
		},
		{
			Labels: map[string]string{
				"name": "good-guy1",
			},
			DependencyInjector: node.NodeDependencyInjector{
				ExecutorsFactory: devstack.NewNoopExecutorsFactoryWithConfig(noop_executor.ExecutorConfig{
					ExternalHooks: noop_executor.ExecutorConfigExternalHooks{
						JobHandler: noop_executor.DelayedJobHandler(goodExecutorSleep),
					},
				}),
			},
		},
		{
			Labels: map[string]string{
				"name": "good-guy2",
			},
			DependencyInjector: node.NodeDependencyInjector{
				ExecutorsFactory: devstack.NewNoopExecutorsFactoryWithConfig(noop_executor.ExecutorConfig{
					ExternalHooks: noop_executor.ExecutorConfigExternalHooks{
						JobHandler: noop_executor.DelayedJobHandler(goodExecutorSleep),
					},
				}),
			},
		},
	}
	ctx := context.Background()

	requesterConfig, err := node.NewRequesterConfigWithDefaults()
	s.Require().NoError(err)
	requesterConfig.OverAskForBidsFactor = 1
	stack := teststack.Setup(ctx, s.T(), fsr, cfg,
		devstack.WithNumberOfRequesterOnlyNodes(1),
		devstack.WithNumberOfComputeOnlyNodes(len(nodeOverrides)-1),
		devstack.WithNodeOverrides(nodeOverrides...),
		devstack.WithRequesterConfig(requesterConfig),
		teststack.WithNoopExecutor(noop_executor.ExecutorConfig{}, cfg.Node.Compute.ManifestCache),
	)

	s.requester = stack.Nodes[0]
	s.client = clientv2.New(fmt.Sprintf("http://%s:%d", s.requester.APIServer.Address, s.requester.APIServer.Port))
	s.Require().NoError(err)
	s.stateResolver = scenario.NewStateResolverFromStore(s.requester.RequesterNode.JobStore)
	nodeutils.WaitForNodeDiscovery(s.T(), s.requester.RequesterNode, len(nodeOverrides))
}

func (s *RetriesSuite) TearDownSuite() {
	if s.requester != nil {
		s.requester.CleanupManager.Cleanup(context.Background())
	}
	for _, n := range s.computeNodes {
		n.CleanupManager.Cleanup(context.Background())
	}
}

func TestRetriesSuite(t *testing.T) {
	suite.Run(t, new(RetriesSuite))
}

func (s *RetriesSuite) TestRetry() {
	testCases := []struct {
		name                    string
		nodes                   []string // nodes to constrain the job to
		concurrency             int
		failed                  bool // whether the job should fail
		expectedJobState        models.JobStateType
		expectedStateCount      IntMatch
		expectedExecutionStates map[models.ExecutionStateType]IntMatch
		expectedExecutionErrors map[models.ExecutionStateType]string
	}{
		{
			name:               "bid-rejected-succeed-with-retry-on-good-nodes",
			nodes:              []string{"bid-rejector", "good-guy1"},
			expectedStateCount: NewIntMatch(2),
			expectedExecutionStates: map[models.ExecutionStateType]IntMatch{
				models.ExecutionStateCompleted:         NewIntMatch(1),
				models.ExecutionStateAskForBidRejected: NewIntMatch(1),
			},
		},
		{
			name:               "bid-rejected-no-good-nodes",
			nodes:              []string{"bid-rejector"},
			failed:             true,
			expectedStateCount: NewIntMatch(1),
			expectedExecutionStates: map[models.ExecutionStateType]IntMatch{
				models.ExecutionStateAskForBidRejected: NewIntMatch(1),
			},
		},
		{
			name:               "execution-failure-succeed-with-retry-on-good-nodes",
			nodes:              []string{"bad-executor", "good-guy1"},
			expectedStateCount: NewIntMatch(2),
			expectedExecutionStates: map[models.ExecutionStateType]IntMatch{
				models.ExecutionStateCompleted: NewIntMatch(1),
				models.ExecutionStateFailed:    NewIntMatch(1),
			},
			expectedExecutionErrors: map[models.ExecutionStateType]string{
				models.ExecutionStateFailed: errExecution.Error(),
			},
		},
		{
			name:               "execution-failure-no-good-nodes",
			nodes:              []string{"bad-executor"},
			failed:             true,
			expectedStateCount: NewIntMatch(1),
			expectedExecutionStates: map[models.ExecutionStateType]IntMatch{
				models.ExecutionStateFailed: NewIntMatch(2), // we retry up to two times on the same node
			},
			expectedExecutionErrors: map[models.ExecutionStateType]string{
				models.ExecutionStateFailed: errExecution.Error(),
			},
		},
		{
			name:               "publish-failure-succeed-with-retry-on-good-nodes",
			nodes:              []string{"bad-publisher", "good-guy1"},
			expectedStateCount: NewIntMatch(2),
			expectedExecutionStates: map[models.ExecutionStateType]IntMatch{
				models.ExecutionStateCompleted: NewIntMatch(1),
				models.ExecutionStateFailed:    NewIntMatch(1),
			},
			expectedExecutionErrors: map[models.ExecutionStateType]string{
				models.ExecutionStateFailed: errPublish.Error(),
			},
		},
		{
			name:               "publish-failure-no-good-nodes",
			nodes:              []string{"bad-publisher"},
			failed:             true,
			expectedStateCount: NewIntMatch(1),
			expectedExecutionStates: map[models.ExecutionStateType]IntMatch{
				models.ExecutionStateFailed: NewIntMatch(2),
			},
			expectedExecutionErrors: map[models.ExecutionStateType]string{
				models.ExecutionStateFailed: errPublish.Error(),
			},
		},
		{
			name:               "publish-partial-failure",
			nodes:              []string{"bad-publisher", "good-guy1"},
			concurrency:        2,
			failed:             true,
			expectedJobState:   models.JobStateTypeFailed,
			expectedStateCount: NewIntMatch(2),
			expectedExecutionStates: map[models.ExecutionStateType]IntMatch{
				// we cancel the good-guy1 if the two attempts on bad-publisher fail but it may
				// have completed so we will allow 0,1 cancelled and 0,1 completed
				models.ExecutionStateCancelled: NewIntMatch(0, 1),
				models.ExecutionStateCompleted: NewIntMatch(0, 1),
				models.ExecutionStateFailed:    NewIntMatch(2), // we retry up to two times on the same node
			},
			expectedExecutionErrors: map[models.ExecutionStateType]string{
				models.ExecutionStateFailed: errPublish.Error(),
			},
		},
		{
			name:               "cancel-slow-executor-on-failure",
			nodes:              []string{"slow-executor", "bad-executor"},
			concurrency:        2,
			failed:             true,
			expectedJobState:   models.JobStateTypeFailed,
			expectedStateCount: NewIntMatch(2),
			expectedExecutionStates: map[models.ExecutionStateType]IntMatch{
				models.ExecutionStateFailed:    NewIntMatch(2), // we retry up to two times on the same node
				models.ExecutionStateCancelled: NewIntMatch(1),
			},
			expectedExecutionErrors: map[models.ExecutionStateType]string{
				models.ExecutionStateFailed: errExecution.Error(),
			},
		},
		{
			name:               "multiple-failures-succeed-with-retry-on-good-nodes",
			nodes:              []string{"bid-rejector", "bad-executor", "good-guy1", "good-guy2"},
			concurrency:        2,
			expectedStateCount: NewIntMatch(3),
			expectedExecutionStates: map[models.ExecutionStateType]IntMatch{
				models.ExecutionStateAskForBidRejected: NewIntMatch(1),
				models.ExecutionStateFailed:            NewIntMatch(1),
				models.ExecutionStateCompleted:         NewIntMatch(2),
			},
			expectedExecutionErrors: map[models.ExecutionStateType]string{
				models.ExecutionStateFailed:    errExecution.Error(),
				models.ExecutionStateCancelled: errExecution.Error(),
			},
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
			ctx := context.Background()

			j := makeBadTargetingJob(s.T(), tc.nodes)
			j.Count = math.Max(1, tc.concurrency)
			j.Normalize()

			submittedJob, err := s.client.Jobs().Put(ctx, &apimodels.PutJobRequest{
				Job: j,
			})
			s.Require().NoError(err)

			if tc.failed {
				s.Require().Error(s.stateResolver.Wait(ctx, submittedJob.JobID, scenario.WaitForSuccessfulCompletion()))
			} else {
				s.Require().NoError(s.stateResolver.Wait(ctx, submittedJob.JobID, scenario.WaitForSuccessfulCompletion()))
			}
			s.Require().NoError(s.stateResolver.Wait(ctx, submittedJob.JobID, scenario.WaitForTerminalStates()))

			jobState, err := s.stateResolver.JobState(ctx, submittedJob.JobID)
			if len(tc.expectedExecutionStates) == 0 {
				// no job state is expected to exist for this scenario
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)

			// verify job state
			if tc.expectedJobState.IsUndefined() {
				if tc.failed {
					tc.expectedJobState = models.JobStateTypeFailed
				} else {
					tc.expectedJobState = models.JobStateTypeCompleted
				}
			}
			s.Require().Equal(tc.expectedJobState, jobState.State.StateType,
				"Expected job in state %s, found %s", tc.expectedJobState.String(), jobState.State.StateType)

			// verify execution states
			executionStates := jobState.GroupExecutionsByState()
			s.Require().True(tc.expectedStateCount.Match(len(executionStates)), "Expected %s, got %d", tc.expectedStateCount.String(), len(executionStates))

			for state, matcher := range tc.expectedExecutionStates {
				// Does the number of states match one of the values in the IntMatch
				s.Require().True(matcher.Match(len(executionStates[state])),
					"Expected %d executions in state %s, found %d", matcher.String(), state.String(), len(executionStates[state]))
			}

			// verify execution error status message
			for state, message := range tc.expectedExecutionErrors {
				for _, execution := range executionStates[state] {
					s.Require().Contains(execution.ComputeState.Message, message)
				}
			}
		})
	}
}

func makeBadTargetingJob(t testing.TB, restrictedNodes []string) *models.Job {
	j := &models.Job{
		Name:  t.Name(),
		Type:  models.JobTypeBatch,
		Count: 1,
		Tasks: []*models.Task{
			{
				Name: t.Name(),
				Engine: &models.SpecConfig{
					Type:   models.EngineNoop,
					Params: make(map[string]interface{}),
				},
				Publisher: &models.SpecConfig{
					Type:   models.PublisherNoop,
					Params: make(map[string]interface{}),
				},
			},
		},
	}
	req := []*models.LabelSelectorRequirement{
		{
			Key:      "favour_name",
			Operator: selection.NotIn,
			Values:   []string{"good-guy1", "good-guy2"},
		}}
	if len(restrictedNodes) > 0 {
		req = append(req, &models.LabelSelectorRequirement{
			Key:      "name",
			Operator: selection.In,
			Values:   restrictedNodes,
		})
	}
	j.Constraints = req
	return j
}

// IntMatch is a type that contains a list of numbers that are
// a possible match. This allows us to say that we want, for
// instance, one item, or zero or one items. This might have
// best been implemented as a Range type, but
type IntMatch struct {
	numbers []int
}

func NewIntMatch(nums ...int) IntMatch {
	return IntMatch{
		numbers: append([]int(nil), nums...),
	}
}

func (i IntMatch) Match(v int) bool {
	return slices.Contains(i.numbers, v)
}

func (i IntMatch) String() string {
	strs := []string{}

	for _, n := range i.numbers {
		strs = append(strs, fmt.Sprintf("%d", n))
	}

	return strings.Join(strs, " OR ")
}

func (s *RetriesSuite) TestMatcher() {
	i := NewIntMatch(0, 1, 2, 3)
	s.Require().Equal(i.String(), "0 OR 1 OR 2 OR 3")
	s.Require().True(i.Match(2))
	s.Require().False(i.Match(10))
}

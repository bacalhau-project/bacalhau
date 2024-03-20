//go:build integration || !unit

package requester

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"
	"golang.org/x/exp/slices"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/migration/legacy"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/test/teststack"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	legacy_job "github.com/bacalhau-project/bacalhau/pkg/legacyjob"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	noop_publisher "github.com/bacalhau-project/bacalhau/pkg/publisher/noop"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
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
	client        *client.APIClient
	stateResolver *legacy_job.StateResolver
}

func (s *RetriesSuite) SetupSuite() {
	logger.ConfigureTestLogging(s.T())

	computeConfig, err := node.NewComputeConfigWith(node.ComputeConfigParams{
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

	requesterConfig, err := node.NewRequesterConfigWith(
		node.RequesterConfigParams{
			NodeRankRandomnessRange: 0,
			OverAskForBidsFactor:    1,
		},
	)
	s.Require().NoError(err)
	stack := teststack.Setup(ctx, s.T(),
		devstack.WithNumberOfRequesterOnlyNodes(1),
		devstack.WithNumberOfComputeOnlyNodes(len(nodeOverrides)-1),
		devstack.WithNodeOverrides(nodeOverrides...),
		devstack.WithRequesterConfig(requesterConfig),
		teststack.WithNoopExecutor(noop_executor.ExecutorConfig{}),
	)

	s.requester = stack.Nodes[0]
	s.client = client.NewAPIClient(client.NoTLS, s.requester.APIServer.Address, s.requester.APIServer.Port)
	s.stateResolver = legacy.NewStateResolver(s.requester.RequesterNode.JobStore)
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
		expectedJobState        model.JobStateType
		expectedExecutionStates map[string]model.ExecutionStateType
		expectedExecutionErrors map[model.ExecutionStateType]string
	}{
		{
			name:  "bid-rejected-succeed-with-retry-on-good-nodes",
			nodes: []string{"bid-rejector", "good-guy1"},
			expectedExecutionStates: map[string]model.ExecutionStateType{
				"good-guy1":    model.ExecutionStateCompleted,
				"bid-rejector": model.ExecutionStateAskForBidRejected,
			},
		},
		{
			name:   "bid-rejected-no-good-nodes",
			nodes:  []string{"bid-rejector"},
			failed: true,
			expectedExecutionStates: map[string]model.ExecutionStateType{
				"bid-rejector": model.ExecutionStateAskForBidRejected,
			},
		},
		{
			name:  "execution-failure-succeed-with-retry-on-good-nodes",
			nodes: []string{"bad-executor", "good-guy1"},
			expectedExecutionStates: map[string]model.ExecutionStateType{
				"good-guy1":    model.ExecutionStateCompleted,
				"bad-executor": model.ExecutionStateFailed,
			},
			expectedExecutionErrors: map[model.ExecutionStateType]string{
				model.ExecutionStateFailed: errExecution.Error(),
			},
		},
		{
			name:   "execution-failure-no-good-nodes",
			nodes:  []string{"bad-executor"},
			failed: true,
			expectedExecutionStates: map[string]model.ExecutionStateType{
				"bad-executor": model.ExecutionStateFailed, // we retry up to two times on the same node
			},
			expectedExecutionErrors: map[model.ExecutionStateType]string{
				model.ExecutionStateFailed: errExecution.Error(),
			},
		},
		{
			name:  "publish-failure-succeed-with-retry-on-good-nodes",
			nodes: []string{"bad-publisher", "good-guy1"},
			expectedExecutionStates: map[string]model.ExecutionStateType{
				"good-guy1":    model.ExecutionStateCompleted,
				"bad-executor": model.ExecutionStateFailed,
			},
			expectedExecutionErrors: map[model.ExecutionStateType]string{
				model.ExecutionStateFailed: errPublish.Error(),
			},
		},
		{
			name:   "publish-failure-no-good-nodes",
			nodes:  []string{"bad-publisher"},
			failed: true,
			expectedExecutionStates: map[string]model.ExecutionStateType{
				"bad-publisher": model.ExecutionStateFailed,
			},
			expectedExecutionErrors: map[model.ExecutionStateType]string{
				model.ExecutionStateFailed: errPublish.Error(),
			},
		},
		{
			name:             "publish-partial-failure",
			nodes:            []string{"bad-publisher", "good-guy1"},
			concurrency:      2,
			failed:           true,
			expectedJobState: model.JobStateError,
			expectedExecutionStates: map[string]model.ExecutionStateType{
				// we cancel the good-guy1 if the two attempts on bad-publisher fail but it may
				// have completed so we will allow 0,1 cancelled and 0,1 completed
				"good-guy1":     model.ExecutionStateCompleted,
				"bad-publisher": model.ExecutionStateFailed, // we retry up to two times on the same node
			},
			expectedExecutionErrors: map[model.ExecutionStateType]string{
				model.ExecutionStateFailed: errPublish.Error(),
			},
		},
		{
			name:             "cancel-slow-executor-on-failure",
			nodes:            []string{"slow-executor", "bad-executor"},
			concurrency:      2,
			failed:           true,
			expectedJobState: model.JobStateError,
			expectedExecutionStates: map[string]model.ExecutionStateType{
				"good-guy1":     model.ExecutionStateFailed, // we retry up to two times on the same node
				"slow-executor": model.ExecutionStateCancelled,
			},
			expectedExecutionErrors: map[model.ExecutionStateType]string{
				model.ExecutionStateFailed:    errExecution.Error(),
				model.ExecutionStateCancelled: "overall job has failed",
			},
		},
		{
			name:        "multiple-failures-succeed-with-retry-on-good-nodes",
			nodes:       []string{"bid-rejector", "bad-executor", "good-guy1", "good-guy2"},
			concurrency: 2,
			expectedExecutionStates: map[string]model.ExecutionStateType{
				"bid-rejector": model.ExecutionStateAskForBidRejected,
				"bad-executor": model.ExecutionStateFailed,
				"good-guy1":    model.ExecutionStateCompleted,
				"good-guy2":    model.ExecutionStateCompleted,
			},
			expectedExecutionErrors: map[model.ExecutionStateType]string{
				model.ExecutionStateFailed:    errExecution.Error(),
				model.ExecutionStateCancelled: errExecution.Error(),
			},
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			ctx := context.Background()
			j := makeBadTargetingJob(s.T(), tc.nodes)
			j.Spec.Deal.Concurrency = math.Max(1, tc.concurrency)
			submittedJob, err := s.client.Submit(ctx, j)
			s.Require().NoError(err)

			if tc.failed {
				s.Require().Error(s.stateResolver.WaitUntilComplete(ctx, submittedJob.ID()))
			} else {
				s.Require().NoError(s.stateResolver.WaitUntilComplete(ctx, submittedJob.ID()))
			}
			s.Require().NoError(s.stateResolver.Wait(ctx, submittedJob.ID(), legacy_job.WaitForTerminalStates()))

			jobState, err := s.stateResolver.GetJobState(ctx, submittedJob.ID())
			if len(tc.expectedExecutionStates) == 0 {
				// no job state is expected to exist for this scenario
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)

			// verify job state
			if tc.expectedJobState.IsUndefined() {
				if tc.failed {
					tc.expectedJobState = model.JobStateError
				} else {
					tc.expectedJobState = model.JobStateCompleted
				}
			}
			s.Require().Equal(tc.expectedJobState, jobState.State,
				"Expected job in state %s, found %s", tc.expectedJobState.String(), jobState.State.String())

			// verify execution states
			nodeInfo, err := s.client.Nodes(ctx)
			s.Require().NoError(err)

			nodes := lo.GroupBy(nodeInfo, func(item models.NodeInfo) string { return item.NodeID })
			executionStates := lo.GroupBy(jobState.Executions, func(s model.ExecutionState) string { return nodes[s.NodeID][0].Labels["name"] })
			stateTypes := lo.MapValues(executionStates, func(states []model.ExecutionState, _ string) []model.ExecutionStateType {
				return lo.Uniq(lo.Map(states, func(item model.ExecutionState, index int) model.ExecutionStateType { return item.State }))
			})
			s.T().Log(stateTypes)
			for node, state := range tc.expectedExecutionStates {
				states := stateTypes[node]
				s.Require().Len(states, 1,
					"Expected node %q to have executions in 1 state %q but has %s states\n%+v", node, state, states, stateTypes)
				s.Require().Contains(states, state,
					"Expected node %q to have executions in state %q but has %s states\n%+v", node, state, states, stateTypes)
			}

			// verify execution error status message
			executionsByState := jobState.GroupExecutionsByState()
			for state, message := range tc.expectedExecutionErrors {
				for _, execution := range executionsByState[state] {
					s.Require().Contains(execution.Status, message)
				}
			}
		})
	}
}

func makeBadTargetingJob(t testing.TB, restrictedNodes []string) *model.Job {
	j := testutils.MakeJobWithOpts(t,
		legacy_job.WithSchedulingTimeout(5),
		legacy_job.WithBaseRetryDelay(1),
	)
	req := []model.LabelSelectorRequirement{
		{
			Key:      "favour_name",
			Operator: selection.NotIn,
			Values:   []string{"good-guy1", "good-guy2"},
		}}
	if len(restrictedNodes) > 0 {
		req = append(req, model.LabelSelectorRequirement{
			Key:      "name",
			Operator: selection.In,
			Values:   restrictedNodes,
		})
	}
	j.Spec.NodeSelectors = req
	return &j
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

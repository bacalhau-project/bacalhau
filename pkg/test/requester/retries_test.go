//go:build integration || !unit

package requester

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	"github.com/bacalhau-project/bacalhau/pkg/test/teststack"

	testing2 "github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	noop_publisher "github.com/bacalhau-project/bacalhau/pkg/publisher/noop"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

var executionErr = errors.New("I am a bad executor")
var publishErr = errors.New("I am a bad publisher")
var slowExecutorSleep = 5 * time.Second
var goodExecutorSleep = 250 * time.Millisecond // force bad executors to finish first to have more predictable tests

type RetriesSuite struct {
	suite.Suite
	requester     *node.Node
	computeNodes  []*node.Node
	client        *publicapi.RequesterAPIClient
	stateResolver *job.StateResolver
}

func (s *RetriesSuite) SetupSuite() {
	logger.ConfigureTestLogging(s.T())
	setup.SetupBacalhauRepoForTesting(s.T())

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
			ComputeConfig: node.NewComputeConfigWith(node.ComputeConfigParams{
				BidSemanticStrategy: testing2.NewFixedBidStrategy(false, false),
				BidResourceStrategy: testing2.NewFixedBidStrategy(false, false),
			}),
		},
		{
			Labels: map[string]string{
				"name": "bad-executor",
			},
			DependencyInjector: node.NodeDependencyInjector{
				ExecutorsFactory: devstack.NewNoopExecutorsFactoryWithConfig(noop_executor.ExecutorConfig{
					ExternalHooks: noop_executor.ExecutorConfigExternalHooks{
						JobHandler: noop_executor.ErrorJobHandler(executionErr),
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
						PublishResult: noop_publisher.ErrorResultPublisher(publishErr),
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

	stack := teststack.Setup(ctx, s.T(),
		devstack.WithNumberOfRequesterOnlyNodes(1),
		devstack.WithNumberOfComputeOnlyNodes(len(nodeOverrides)-1),
		devstack.WithNodeOverrides(nodeOverrides...),
		devstack.WithRequesterConfig(
			node.NewRequesterConfigWith(
				node.RequesterConfigParams{
					NodeRankRandomnessRange: 0,
					OverAskForBidsFactor:    1,
				},
			),
		),
		teststack.WithNoopExecutor(noop_executor.ExecutorConfig{}),
	)

	s.requester = stack.Nodes[0]
	s.client = publicapi.NewRequesterAPIClient(s.requester.APIServer.Address, s.requester.APIServer.Port)
	s.stateResolver = job.NewStateResolver(
		func(ctx context.Context, id string) (model.Job, error) {
			return s.requester.RequesterNode.JobStore.GetJob(ctx, id)
		},
		func(ctx context.Context, id string) (model.JobState, error) {
			return s.requester.RequesterNode.JobStore.GetJobState(ctx, id)
		},
	)
	testutils.WaitForNodeDiscovery(s.T(), s.requester, len(nodeOverrides))
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
		expectedExecutionStates map[model.ExecutionStateType]int
		expectedExecutionErrors map[model.ExecutionStateType]string
	}{
		{
			name:  "bid-rejected-succeed-with-retry-on-good-nodes",
			nodes: []string{"bid-rejector", "good-guy1"},
			expectedExecutionStates: map[model.ExecutionStateType]int{
				model.ExecutionStateCompleted:         1,
				model.ExecutionStateAskForBidRejected: 1,
			},
		},
		{
			name:   "bid-rejected-no-good-nodes",
			nodes:  []string{"bid-rejector"},
			failed: true,
			expectedExecutionStates: map[model.ExecutionStateType]int{
				model.ExecutionStateAskForBidRejected: 1,
			},
		},
		{
			name:  "execution-failure-succeed-with-retry-on-good-nodes",
			nodes: []string{"bad-executor", "good-guy1"},
			expectedExecutionStates: map[model.ExecutionStateType]int{
				model.ExecutionStateCompleted: 1,
				model.ExecutionStateFailed:    1,
			},
			expectedExecutionErrors: map[model.ExecutionStateType]string{
				model.ExecutionStateFailed: executionErr.Error(),
			},
		},
		{
			name:   "execution-failure-no-good-nodes",
			nodes:  []string{"bad-executor"},
			failed: true,
			expectedExecutionStates: map[model.ExecutionStateType]int{
				model.ExecutionStateFailed: 2, // we retry up to two times on the same node
			},
			expectedExecutionErrors: map[model.ExecutionStateType]string{
				model.ExecutionStateFailed: executionErr.Error(),
			},
		},
		{
			name:  "publish-failure-succeed-with-retry-on-good-nodes",
			nodes: []string{"bad-publisher", "good-guy1"},
			expectedExecutionStates: map[model.ExecutionStateType]int{
				model.ExecutionStateCompleted: 1,
				model.ExecutionStateFailed:    1,
			},
			expectedExecutionErrors: map[model.ExecutionStateType]string{
				model.ExecutionStateFailed: publishErr.Error(),
			},
		},
		{
			name:   "publish-failure-no-good-nodes",
			nodes:  []string{"bad-publisher"},
			failed: true,
			expectedExecutionStates: map[model.ExecutionStateType]int{
				model.ExecutionStateFailed: 2,
			},
			expectedExecutionErrors: map[model.ExecutionStateType]string{
				model.ExecutionStateFailed: publishErr.Error(),
			},
		},
		{
			name:             "publish-partial-failure",
			nodes:            []string{"bad-publisher", "good-guy1"},
			concurrency:      2,
			failed:           true,
			expectedJobState: model.JobStateError,
			expectedExecutionStates: map[model.ExecutionStateType]int{
				model.ExecutionStateCancelled: 1, // we cancel the good-guy1 if the two attempts on bad-publisher fail
				model.ExecutionStateFailed:    2, // we retry up to two times on the same node
			},
			expectedExecutionErrors: map[model.ExecutionStateType]string{
				model.ExecutionStateFailed: publishErr.Error(),
			},
		},
		{
			name:             "cancel-slow-executor-on-failure",
			nodes:            []string{"slow-executor", "bad-executor"},
			concurrency:      2,
			failed:           true,
			expectedJobState: model.JobStateError,
			expectedExecutionStates: map[model.ExecutionStateType]int{
				model.ExecutionStateFailed:    2, // we retry up to two times on the same node
				model.ExecutionStateCancelled: 1,
			},
			expectedExecutionErrors: map[model.ExecutionStateType]string{
				model.ExecutionStateFailed:    executionErr.Error(),
				model.ExecutionStateCancelled: "overall job has failed",
			},
		},
		{
			name:        "multiple-failures-succeed-with-retry-on-good-nodes",
			nodes:       []string{"bid-rejector", "bad-executor", "good-guy1", "good-guy2"},
			concurrency: 2,
			expectedExecutionStates: map[model.ExecutionStateType]int{
				model.ExecutionStateAskForBidRejected: 1,
				model.ExecutionStateFailed:            1,
				model.ExecutionStateCompleted:         2,
			},
			expectedExecutionErrors: map[model.ExecutionStateType]string{
				model.ExecutionStateFailed:    executionErr.Error(),
				model.ExecutionStateCancelled: executionErr.Error(),
			},
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			ctx := context.Background()
			j := makeBadTargetingJob(tc.nodes)
			j.Spec.Deal.Concurrency = math.Max(1, tc.concurrency)
			submittedJob, err := s.client.Submit(ctx, j)
			if tc.failed {
				s.Error(s.stateResolver.WaitUntilComplete(ctx, submittedJob.ID()))
			} else {
				s.NoError(s.stateResolver.WaitUntilComplete(ctx, submittedJob.ID()))
			}
			s.NoError(s.stateResolver.Wait(ctx, submittedJob.ID(), job.WaitForTerminalStates()))

			jobState, err := s.stateResolver.GetJobState(ctx, submittedJob.ID())
			if len(tc.expectedExecutionStates) == 0 {
				// no job state is expected to exist for this scenario
				s.Error(err)
				return
			}
			s.NoError(err)

			// verify job state
			if tc.expectedJobState.IsUndefined() {
				if tc.failed {
					tc.expectedJobState = model.JobStateError
				} else {
					tc.expectedJobState = model.JobStateCompleted
				}
			}
			s.Equal(tc.expectedJobState, jobState.State,
				"Expected job in state %s, found %s", tc.expectedJobState.String(), jobState.State.String())

			// verify execution states
			executionStates := jobState.GroupExecutionsByState()
			s.Equal(len(tc.expectedExecutionStates), len(executionStates))
			for state, count := range tc.expectedExecutionStates {
				s.Equal(count, len(executionStates[state]),
					"Expected %d executions in state %s, found %d", count, state.String(), len(executionStates[state]))
			}

			// verify execution error status message
			for state, message := range tc.expectedExecutionErrors {
				for _, execution := range executionStates[state] {
					s.Contains(execution.Status, message)
				}
			}
		})
	}
}

func makeBadTargetingJob(restrictedNodes []string) *model.Job {
	j := testutils.MakeJob(model.EngineNoop, model.PublisherNoop, []string{"echo", "hello"})
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
	return j
}

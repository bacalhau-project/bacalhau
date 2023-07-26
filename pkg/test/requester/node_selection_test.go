//go:build integration || !unit

package requester

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/labels"
)

type NodeSelectionSuite struct {
	suite.Suite
	requester     *node.Node
	compute1      *node.Node
	compute2      *node.Node
	compute3      *node.Node
	computeNodes  []*node.Node
	client        *publicapi.RequesterAPIClient
	stateResolver *job.StateResolver
}

func (s *NodeSelectionSuite) SetupSuite() {
	logger.ConfigureTestLogging(s.T())
	system.InitConfigForTesting(s.T())

	ctx := context.Background()
	devstackOptions := devstack.DevStackOptions{
		NumberOfRequesterOnlyNodes: 1,
		NumberOfComputeOnlyNodes:   3,
	}

	nodeOverrides := []node.NodeConfig{
		{}, // pass overriding requester node
		{
			Labels: map[string]string{
				"name": "compute-1",
				"env":  "prod",
			},
		},
		{
			Labels: map[string]string{
				"name": "compute-2",
				"env":  "prod",
			},
		},
		{
			Labels: map[string]string{
				"name": "compute-3",
				"env":  "test",
			},
		},
	}
	stack := testutils.SetupTestWithNoopExecutor(ctx, s.T(), devstackOptions,
		node.NewComputeConfigWithDefaults(),
		node.NewRequesterConfigWith(node.RequesterConfigParams{
			NodeRankRandomnessRange: 0,
			OverAskForBidsFactor:    1,
		}),
		noop_executor.ExecutorConfig{},
		nodeOverrides...,
	)

	s.requester = stack.Nodes[0]
	s.compute1 = stack.Nodes[1]
	s.compute2 = stack.Nodes[2]
	s.compute3 = stack.Nodes[3]
	s.client = publicapi.NewRequesterAPIClient(s.requester.APIServer.Address, s.requester.APIServer.Port)
	s.stateResolver = job.NewStateResolver(
		func(ctx context.Context, id string) (model.Job, error) {
			return s.requester.RequesterNode.JobStore.GetJob(ctx, id)
		},
		func(ctx context.Context, id string) (model.JobState, error) {
			return s.requester.RequesterNode.JobStore.GetJobState(ctx, id)
		},
	)
	s.computeNodes = []*node.Node{s.compute1, s.compute2, s.compute3}

	testutils.WaitForNodeDiscovery(s.T(), s.requester, 4)
}

func (s *NodeSelectionSuite) TearDownSuite() {
	if s.requester != nil {
		s.requester.CleanupManager.Cleanup(context.Background())
	}
	for _, n := range s.computeNodes {
		n.CleanupManager.Cleanup(context.Background())
	}
}

func TestNodeSelectionSuite(t *testing.T) {
	suite.Run(t, new(NodeSelectionSuite))
}

func (s *NodeSelectionSuite) TestNodeSelectionByLabels() {

	testCase := []struct {
		name          string
		selector      string
		expectedNodes []*node.Node
	}{
		{
			name:          "select by name",
			selector:      "name=compute-1",
			expectedNodes: []*node.Node{s.compute1},
		},
		{
			name:          "select by env",
			selector:      "env=prod",
			expectedNodes: []*node.Node{s.compute1, s.compute2},
		},
		{
			name:          "select by name and env",
			selector:      "name=compute-1,env=prod",
			expectedNodes: []*node.Node{s.compute1},
		},
		{
			name:          "select by negated env",
			selector:      "env!=prod",
			expectedNodes: []*node.Node{s.compute3},
		},
		{
			name:          "select by multiple env",
			selector:      "env in (prod,test)",
			expectedNodes: []*node.Node{s.compute1, s.compute2, s.compute3},
		},
		{
			name:          "select by multiple negative env",
			selector:      "env notin (prod,test)",
			expectedNodes: []*node.Node{},
		},
		{
			name:          "favour by name",
			selector:      "favour_name=compute-1,name in (compute-1,compute-2)",
			expectedNodes: []*node.Node{s.compute1}, // concurrency=1
		},
		{
			name:          "favour by name multiple nodes",
			selector:      "favour_name=compute-1,env=prod",
			expectedNodes: []*node.Node{s.compute1, s.compute2}, // concurrency=2
		},
		{
			name:          "favour by name multiple nodes",
			selector:      "favour_name=compute-1,env=prod",
			expectedNodes: []*node.Node{s.compute1, s.compute2}, // concurrency=2
		},
	}

	for _, tc := range testCase {
		s.Run(tc.name, func() {
			ctx := context.Background()
			j := testutils.MakeNoopJob()
			j.Spec.NodeSelectors = s.parseLabels(tc.selector)
			j.Spec.Deal.Concurrency = math.Max(1, len(tc.expectedNodes))

			submittedJob, err := s.client.Submit(ctx, j)
			s.NoError(err)

			err = s.stateResolver.WaitUntilComplete(ctx, submittedJob.Metadata.ID)
			if len(tc.expectedNodes) == 0 {
				s.Error(err)
			} else {
				s.NoError(err)
			}
			selectedNodes := s.getSelectedNodes(submittedJob.Metadata.ID)
			s.assertNodesMatch(tc.expectedNodes, selectedNodes)
		})
	}
}

func (s *NodeSelectionSuite) getSelectedNodes(jobID string) []*node.Node {
	ctx := context.Background()
	jobState, err := s.stateResolver.GetJobState(ctx, jobID)
	s.NoError(err)
	completedExecutionStates := job.GetCompletedExecutionStates(jobState)

	nodes := make([]*node.Node, 0, len(completedExecutionStates))
	for _, executionState := range completedExecutionStates {
		nodeFound := false
		for _, n := range s.computeNodes {
			if n.Host.ID().String() == executionState.NodeID {
				nodes = append(nodes, n)
				nodeFound = true
				break
			}
		}
		if !nodeFound {
			s.Fail("node not found", executionState.NodeID)
		}
	}
	return nodes
}

func (s *NodeSelectionSuite) assertNodesMatch(expected, selected []*node.Node) {
	expectedNodeNames := make([]string, 0, len(expected))
	selectedNodeNames := make([]string, 0, len(selected))
	for _, n := range expected {
		expectedNodeNames = append(expectedNodeNames, n.Host.ID().String())
	}
	for _, n := range selected {
		selectedNodeNames = append(selectedNodeNames, n.Host.ID().String())
	}
	s.ElementsMatch(expectedNodeNames, selectedNodeNames)
}

func (s *NodeSelectionSuite) parseLabels(selector string) []model.LabelSelectorRequirement {
	requirements, err := labels.ParseToRequirements(selector)
	s.NoError(err)
	return model.ToLabelSelectorRequirements(requirements...)
}

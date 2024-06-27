//go:build integration || !unit

package requester

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	"github.com/bacalhau-project/bacalhau/pkg/test/teststack"
	nodeutils "github.com/bacalhau-project/bacalhau/pkg/test/utils/node"
)

type NodeSelectionSuite struct {
	suite.Suite
	requester     *node.Node
	compute1      *node.Node
	compute2      *node.Node
	compute3      *node.Node
	computeNodes  []*node.Node
	api           clientv2.API
	stateResolver *scenario.StateResolver
}

func (s *NodeSelectionSuite) SetupSuite() {
	logger.ConfigureTestLogging(s.T())
	fsr, cfg := setup.SetupBacalhauRepoForTesting(s.T())

	ctx := context.Background()

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
	requesterConfig, err := node.NewRequesterConfigWithDefaults()
	s.Require().NoError(err)
	requesterConfig.OverAskForBidsFactor = 1
	stack := teststack.Setup(ctx, s.T(), fsr, cfg,
		devstack.WithNumberOfRequesterOnlyNodes(1),
		devstack.WithNumberOfComputeOnlyNodes(3),
		devstack.WithNodeOverrides(nodeOverrides...),
		devstack.WithRequesterConfig(requesterConfig),
		teststack.WithNoopExecutor(noop_executor.ExecutorConfig{}, cfg.Node.Compute.ManifestCache),
	)

	s.requester = stack.Nodes[0]
	s.compute1 = stack.Nodes[1]
	s.compute2 = stack.Nodes[2]
	s.compute3 = stack.Nodes[3]
	s.api = clientv2.New(fmt.Sprintf("http://%s:%d", s.requester.APIServer.Address, s.requester.APIServer.Port))
	s.Require().NoError(err)
	s.stateResolver = scenario.NewStateResolver(s.api)
	s.computeNodes = []*node.Node{s.compute1, s.compute2, s.compute3}

	nodeutils.WaitForNodeDiscovery(s.T(), s.requester.RequesterNode, 4)
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
			j := &models.Job{
				Name:  s.T().Name(),
				Type:  models.JobTypeBatch,
				Count: math.Max(1, len(tc.expectedNodes)),
				Tasks: []*models.Task{
					{
						Name: s.T().Name(),
						Engine: &models.SpecConfig{
							Type:   models.EngineNoop,
							Params: make(map[string]interface{}),
						},
					},
				},
			}
			j.Constraints = s.parseLabels(tc.selector)
			j.Normalize()

			putResp, err := s.api.Jobs().Put(ctx, &apimodels.PutJobRequest{Job: j})
			s.NoError(err)

			err = s.stateResolver.Wait(ctx, putResp.JobID, scenario.WaitForSuccessfulCompletion())
			if len(tc.expectedNodes) == 0 {
				s.Error(err)
			} else {
				s.NoError(err)
			}
			selectedNodes := s.getSelectedNodes(putResp.JobID)
			s.assertNodesMatch(tc.expectedNodes, selectedNodes)
		})
	}
}

func (s *NodeSelectionSuite) getSelectedNodes(jobID string) []*node.Node {
	ctx := context.Background()
	jobState, err := s.stateResolver.JobState(ctx, jobID)
	s.NoError(err)
	completedExecutionStates := scenario.GetCompletedExecutionStates(jobState)

	nodes := make([]*node.Node, 0, len(completedExecutionStates))
	for _, executionState := range completedExecutionStates {
		nodeFound := false
		for _, n := range s.computeNodes {
			if n.ID == executionState.NodeID {
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
		expectedNodeNames = append(expectedNodeNames, n.ID)
	}
	for _, n := range selected {
		selectedNodeNames = append(selectedNodeNames, n.ID)
	}
	s.ElementsMatch(expectedNodeNames, selectedNodeNames)
}

func (s *NodeSelectionSuite) parseLabels(selector string) []*models.LabelSelectorRequirement {
	requirements, err := labels.ParseToRequirements(selector)
	s.NoError(err)
	tmp := models.ToLabelSelectorRequirements(requirements...)
	out := make([]*models.LabelSelectorRequirement, 0, len(tmp))
	for _, r := range tmp {
		out = append(out, r.Copy())
	}
	return out
}

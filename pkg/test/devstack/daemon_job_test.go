//go:build integration || !unit

package devstack

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	dockmodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/test/teststack"
)

type DaemonJobSuite struct {
	suite.Suite
	stack  *devstack.DevStack
	client clientv2.API
	ctx    context.Context
	cm     *system.CleanupManager
}

func TestDevstackDaemonJobSuite(t *testing.T) {
	suite.Run(t, new(DaemonJobSuite))
}

func (s *DaemonJobSuite) SetupSuite() {
	docker.MustHaveDocker(s.T())
	s.ctx = context.Background()
	s.cm = system.NewCleanupManager()

	// Create the devstack cluster
	s.stack = teststack.Setup(
		s.ctx,
		s.T(),
		devstack.WithNumberOfHybridNodes(1), // Need at least one orchestrator
		devstack.WithNumberOfComputeOnlyNodes(3),
		devstack.WithNodeOverrides([]node.NodeConfig{
			{}, // Hybrid node - no specific config
			{
				BacalhauConfig: types.Bacalhau{
					Labels: map[string]string{
						"zone": "us-east-1",
						"type": "worker",
					},
				},
			},
			{
				BacalhauConfig: types.Bacalhau{
					Labels: map[string]string{
						"zone": "us-west-1",
						"type": "worker",
					},
				},
			},
			{
				BacalhauConfig: types.Bacalhau{
					Labels: map[string]string{
						"zone": "eu-west-1",
						"type": "storage", // Different type - should not match
					},
				},
			},
		}...),
	)
	s.client = clientv2.New(fmt.Sprintf("http://%s:%d", s.stack.Nodes[0].APIServer.Address, s.stack.Nodes[0].APIServer.Port))
}

func (s *DaemonJobSuite) TearDownSuite() {
	if s.cm != nil {
		s.cm.Cleanup(s.ctx)
	}
}

// verifyExecutionsState waits until the execution states per node match expected states and no extra executions exist
func (s *DaemonJobSuite) verifyExecutionsState(jobID string, expectedNodeStates map[string]models.ExecutionStateType) {
	var lastErrorMsg string
	var lastObserved map[string]models.ExecutionStateType

	s.Eventually(func() bool {
		getResp, err := s.client.Jobs().Get(s.ctx, &apimodels.GetJobRequest{
			JobIDOrName: jobID,
			Include:     "executions",
		})
		if err != nil || getResp.Executions == nil {
			lastErrorMsg = "Failed to get executions or executions missing"
			lastObserved = nil
			return false
		}

		actualNodeStates := make(map[string]models.ExecutionStateType)
		for _, exec := range getResp.Executions.Items {
			actualNodeStates[exec.NodeID] = exec.ComputeState.StateType
		}
		lastObserved = actualNodeStates

		var errorMsg string
		// Check all expected nodes have the correct state
		for nodeID, expectedState := range expectedNodeStates {
			actualState, found := actualNodeStates[nodeID]
			if !found {
				errorMsg += fmt.Sprintf("Expected execution for node %s not found. ", nodeID)
				continue
			}
			if actualState != expectedState {
				errorMsg += fmt.Sprintf("Execution for node %s: expected %s, got %s. ", nodeID, expectedState, actualState)
			}
		}
		// Check there are no unexpected executions
		if len(actualNodeStates) != len(expectedNodeStates) {
			for nodeID := range actualNodeStates {
				if _, ok := expectedNodeStates[nodeID]; !ok {
					errorMsg += fmt.Sprintf("Unexpected execution found for node %s. ", nodeID)
				}
			}
		}
		lastErrorMsg = errorMsg
		return errorMsg == ""
	}, 5*time.Second, 50*time.Millisecond, "Executions did not match expected states. See test log for details.")

	if lastErrorMsg != "" {
		s.Require().Failf("Execution state verification failed",
			"Last error: %s. Observed states: %v", lastErrorMsg, lastObserved)
	}
}

// submitDaemonJob creates and submits a daemon job
func (s *DaemonJobSuite) submitDaemonJob() string {
	daemonJob := &models.Job{
		Name: uuid.New().String(),
		Type: models.JobTypeDaemon,
		Tasks: []*models.Task{
			{
				Name: s.T().Name(),
				Engine: dockmodels.NewDockerEngineBuilder("busybox:1.37.0").
					WithEntrypoint("sh", "-c", "while true; do echo 'daemon running'; sleep 10; done").
					MustBuild(),
			},
		},
		Constraints: []*models.LabelSelectorRequirement{
			{
				Key:      "type",
				Operator: selection.Equals,
				Values:   []string{"worker"},
			},
		},
	}

	submitResp, err := s.client.Jobs().Put(s.ctx, &apimodels.PutJobRequest{
		Job: daemonJob,
	})
	s.Require().NoError(err)
	return submitResp.JobID
}

// TestDaemonJobNodeJoining tests dynamic node joining behavior with daemon jobs
func (s *DaemonJobSuite) TestDaemonJobNodeJoining() {
	var jobID string
	defer func() {
		if jobID != "" {
			// Ensure we stop the job at the end of the test
			_, _ = s.client.Jobs().Stop(s.ctx, &apimodels.StopJobRequest{JobID: jobID})
		}
	}()

	s.Run("Deploy daemon job and verify 2 executions created", func() {
		jobID = s.submitDaemonJob()

		expectedStates := map[string]models.ExecutionStateType{
			"node-1": models.ExecutionStateNew,
			"node-2": models.ExecutionStateNew,
		}
		s.verifyExecutionsState(jobID, expectedStates)
	})

	s.Run("Stop node and verify its execution is stopped", func() {
		// Stop one of the worker nodes
		err := s.stack.Nodes[1].Stop(context.Background())
		s.Require().NoError(err)

		// Verify remaining node still has running execution
		expectedStates := map[string]models.ExecutionStateType{
			"node-1": models.ExecutionStateFailed,
			"node-2": models.ExecutionStateNew,
		}
		s.verifyExecutionsState(jobID, expectedStates)
	})

	s.Run("Join 2 new nodes and verify only matching node gets execution", func() {
		// Add node with non-matching label
		nonMatchingNode, err := s.stack.JoinNode(s.ctx, s.cm, devstack.JoinNodeOptions{
			IsCompute: true,
			ConfigOverride: &node.NodeConfig{
				BacalhauConfig: types.Bacalhau{
					Labels: map[string]string{
						"zone": "ap-south-2",
						"type": "database", // Does not match daemon job constraint
					},
				},
			},
		})
		s.Require().NoError(err)
		s.Require().NotNil(nonMatchingNode)

		// Add node with matching label
		matchingNode, err := s.stack.JoinNode(s.ctx, s.cm, devstack.JoinNodeOptions{
			IsCompute: true,
			ConfigOverride: &node.NodeConfig{
				BacalhauConfig: types.Bacalhau{
					Labels: map[string]string{
						"zone": "ap-south-1",
						"type": "worker", // Matches daemon job constraint
					},
				},
			},
		})
		s.Require().NoError(err)
		s.Require().NotNil(matchingNode)

		// Verify the new nodes are discovered
		teststack.AllNodesDiscovered(s.T(), s.stack)

		// Final verification: exactly 3 running executions on all worker nodes
		expectedStates := map[string]models.ExecutionStateType{
			"node-1": models.ExecutionStateFailed,
			"node-2": models.ExecutionStateNew,
			"node-5": models.ExecutionStateNew, // New matching node
		}
		s.verifyExecutionsState(jobID, expectedStates)
	})

	// Stop the job at the end to ensure cleanup
	s.Run("Stop daemon job", func() {
		_, err := s.client.Jobs().Stop(s.ctx, &apimodels.StopJobRequest{JobID: jobID})
		s.Require().NoError(err, "Failed to stop daemon job at test end")

		expectedStates := map[string]models.ExecutionStateType{
			"node-1": models.ExecutionStateFailed,
			"node-2": models.ExecutionStateCancelled,
			"node-5": models.ExecutionStateCancelled,
		}
		s.verifyExecutionsState(jobID, expectedStates)
	})
}

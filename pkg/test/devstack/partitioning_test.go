//go:build integration || !unit

package devstack

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	dockmodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	"github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	publisher_local "github.com/bacalhau-project/bacalhau/pkg/publisher/local"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
)

type PartitionSuite struct {
	scenario.ScenarioRunner
}

func TestDevstackPartitionSuite(t *testing.T) {
	suite.Run(t, new(PartitionSuite))
}

func (s *PartitionSuite) SetupSuite() {
	docker.MustHaveDocker(s.T())
}

// TestSinglePartition verifies that a job with a single partition
// gets the correct partition index and job-related environment variables
func (s *PartitionSuite) TestSinglePartition() {
	testCase := scenario.Scenario{
		Stack: &scenario.StackConfig{
			DevStackOptions: []devstack.ConfigOption{
				devstack.WithNumberOfHybridNodes(1),
			},
		},
		Job: &models.Job{
			Name:  s.T().Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name: s.T().Name(),
					Engine: dockmodels.NewDockerEngineBuilder("busybox:1.37.0").
						WithEntrypoint("sh", "-c", "printenv | grep BACALHAU_ | sort").
						MustBuild(),
					Publisher: publisher_local.NewSpecConfig(),
				},
			},
		},
		ResultsChecker: scenario.FileContains(
			"stdout",
			[]string{
				"BACALHAU_EXECUTION_ID=",
				"BACALHAU_JOB_ID=",
				"BACALHAU_JOB_NAME=" + s.T().Name(),
				"BACALHAU_JOB_NAMESPACE=default",
				"BACALHAU_JOB_TYPE=batch",
				"BACALHAU_NODE_ID=node-0",
				"BACALHAU_PARTITION_COUNT=1",
				"BACALHAU_PARTITION_INDEX=0",
			},
			-1, // Don't check line count
		),
		JobCheckers: []scenario.StateChecks{
			scenario.WaitForSuccessfulCompletion(),
		},
	}

	s.RunScenario(testCase)
}

// TestMultiplePartitions verifies that a job with multiple partitions
// assigns unique partition indices across executions
func (s *PartitionSuite) TestMultiplePartitions() {
	testCase := scenario.Scenario{
		Stack: &scenario.StackConfig{
			DevStackOptions: []devstack.ConfigOption{
				devstack.WithNumberOfHybridNodes(1),
				devstack.WithNumberOfComputeOnlyNodes(2),
			},
		},
		Job: &models.Job{
			Name:  s.T().Name(),
			Type:  models.JobTypeBatch,
			Count: 3, // Create 3 partitions
			Tasks: []*models.Task{
				{
					Name: s.T().Name(),
					Engine: dockmodels.NewDockerEngineBuilder("busybox:1.37.0").
						WithEntrypoint("sh", "-c",
							"echo Node=${BACALHAU_NODE_ID} Partition=${BACALHAU_PARTITION_INDEX} of ${BACALHAU_PARTITION_COUNT}",
						).
						MustBuild(),
					Publisher: publisher_local.NewSpecConfig(),
				},
			},
		},
		ResultsChecker: scenario.ManyChecks(
			// Verify all partitions run
			scenario.FileContains(
				"stdout",
				[]string{
					"Partition=0 of 3",
					"Partition=1 of 3",
					"Partition=2 of 3",
				},
				-1,
			),
			// Verify they run on the nodes
			scenario.FileContains(
				"stdout",
				[]string{
					"Node=node-0",
					"Node=node-1",
					"Node=node-2",
				},
				-1,
			),
		),
		JobCheckers: []scenario.StateChecks{
			scenario.WaitForSuccessfulCompletion(),
			scenario.WaitForExecutionStates(map[models.ExecutionStateType]int{
				models.ExecutionStateCompleted: 3,
			}),
		},
	}

	s.RunScenario(testCase)
}

// TestPartitionRetry verifies that when a partition fails, it is retried
// with the same partition index but potentially on a different node
// while other partitions remain unaffected
func (s *PartitionSuite) TestPartitionRetry() {
	var attempt atomic.Int32
	const failedAttempts = 3

	testCase := scenario.Scenario{
		Stack: &scenario.StackConfig{
			DevStackOptions: []devstack.ConfigOption{
				devstack.WithNumberOfHybridNodes(1),
				devstack.WithNumberOfComputeOnlyNodes(3), // More nodes for multiple retries
			},
			ExecutorConfig: noop.ExecutorConfig{
				ExternalHooks: noop.ExecutorConfigExternalHooks{
					JobHandler: func(ctx context.Context, execContext noop.ExecutionContext) (*models.RunCommandResult, error) {
						partition := execContext.Env["BACALHAU_PARTITION_INDEX"]
						nodeID := execContext.Env["BACALHAU_NODE_ID"]

						output := fmt.Sprintf("Running partition %s on node %s\n", partition, nodeID)

						// Only increment attempt counter for partition 1
						var currentAttempt int32
						if partition == "1" {
							currentAttempt = attempt.Add(1)
							output += fmt.Sprintf("Attempt %d for partition 1\n", currentAttempt)

							if currentAttempt < failedAttempts {
								output += fmt.Sprintf("Failing partition 1 on attempt %d\n", currentAttempt)
								// Create result with output but return error
								result := &models.RunCommandResult{
									STDOUT:   output,
									ErrorMsg: fmt.Sprintf("simulated failure on attempt %d", currentAttempt),
								}
								return result, fmt.Errorf("simulated failure")
							}
						}

						output += fmt.Sprintf("Success on partition %s\n", partition)
						return executor.WriteJobResults(execContext.ResultsDir,
							strings.NewReader(output),
							nil,
							0,
							nil,
							executor.OutputLimits{
								MaxStdoutFileLength:   system.MaxStdoutFileLength,
								MaxStdoutReturnLength: system.MaxStdoutReturnLength,
								MaxStderrFileLength:   system.MaxStderrFileLength,
								MaxStderrReturnLength: system.MaxStderrReturnLength,
							}), nil
					},
				},
			},
		},
		Job: &models.Job{
			Name:  s.T().Name(),
			Type:  models.JobTypeBatch,
			Count: 2,
			Tasks: []*models.Task{
				{
					Name: s.T().Name(),
					Engine: &models.SpecConfig{
						Type:   models.EngineNoop,
						Params: make(map[string]interface{}),
					},
					Publisher: publisher_local.NewSpecConfig(),
				},
			},
		},
		ResultsChecker: scenario.FileContains(
			"stdout",
			[]string{
				"Running partition 0",
				"Success on partition 0",
				"Running partition 1",
				"Attempt 3 for partition 1",
				"Success on partition 1",
			},
			-1,
		),
		JobCheckers: []scenario.StateChecks{
			scenario.WaitForSuccessfulCompletion(),
			scenario.WaitForExecutionStates(map[models.ExecutionStateType]int{
				models.ExecutionStateCompleted: 2,
				models.ExecutionStateFailed:    2,
			}),
		},
	}

	s.RunScenario(testCase)
}

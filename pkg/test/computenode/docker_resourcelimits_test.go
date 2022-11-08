//go:build !windows && !(unit && darwin)

package computenode

import (
	"context"
	"strconv"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/capacitymanager"
	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/model"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/stretchr/testify/require"
)

func (suite *ComputeNodeResourceLimitsSuite) TestDockerResourceLimitsCPU() {
	ctx := context.Background()
	CPU_LIMIT := "100m"

	stack := testutils.NewDevStack(ctx, suite.T(), computenode.NewDefaultComputeNodeConfig())
	computeNode, cm := stack.Node.ComputeNode, stack.Node.CleanupManager
	defer cm.Cleanup()

	// this will give us a numerator and denominator that should end up at the
	// same 0.1 value that 100m means
	// https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/8/html/managing_monitoring_and_updating_the_kernel/using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications_managing-monitoring-and-updating-the-kernel#proc_controlling-distribution-of-cpu-time-for-applications-by-adjusting-cpu-bandwidth_using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications
	result := RunJobGetStdout(ctx, suite.T(), computeNode, model.Spec{
		Engine:   model.EngineDocker,
		Verifier: model.VerifierNoop,
		Resources: model.ResourceUsageConfig{
			CPU:    CPU_LIMIT,
			Memory: "100mb",
		},
		Docker: model.JobSpecDocker{
			Image: "ubuntu",
			Entrypoint: []string{
				"bash",
				"-c",
				"cat /sys/fs/cgroup/cpu.max",
			},
		},
	})

	values := strings.Fields(result)

	numerator, err := strconv.Atoi(values[0])
	require.NoError(suite.T(), err)

	denominator, err := strconv.Atoi(values[1])
	require.NoError(suite.T(), err)

	var containerCPU float64 = 0

	if denominator > 0 {
		containerCPU = float64(numerator) / float64(denominator)
	}

	require.Equal(suite.T(), capacitymanager.ConvertCPUString(CPU_LIMIT), containerCPU, "the container reported CPU does not equal the configured limit")
}

func (suite *ComputeNodeResourceLimitsSuite) TestDockerResourceLimitsMemory() {
	ctx := context.Background()
	MEMORY_LIMIT := "100mb"

	stack := testutils.NewDevStack(ctx, suite.T(), computenode.NewDefaultComputeNodeConfig())
	computeNode, cm := stack.Node.ComputeNode, stack.Node.CleanupManager
	defer cm.Cleanup()

	result := RunJobGetStdout(ctx, suite.T(), computeNode, model.Spec{
		Engine:   model.EngineDocker,
		Verifier: model.VerifierNoop,
		Resources: model.ResourceUsageConfig{
			CPU:    "100m",
			Memory: MEMORY_LIMIT,
		},
		Docker: model.JobSpecDocker{
			Image: "ubuntu",
			Entrypoint: []string{
				"bash",
				"-c",
				"cat /sys/fs/cgroup/memory.max",
			},
		},
	})

	intVar, err := strconv.Atoi(strings.TrimSpace(result))
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), capacitymanager.ConvertMemoryString(MEMORY_LIMIT), uint64(intVar), "the container reported memory does not equal the configured limit")
}

func (suite *ComputeNodeResourceLimitsSuite) TestDockerResourceLimitsDisk() {
	ctx := context.Background()

	runTest := func(text, diskSize string, expected bool) {
		stack := testutils.NewDevStack(ctx, suite.T(), computenode.ComputeNodeConfig{
			CapacityManagerConfig: capacitymanager.Config{
				ResourceLimitTotal: model.ResourceUsageConfig{
					// so we have a compute node with 1 byte of disk space
					Disk: diskSize,
				},
			},
		})
		computeNode, ipfsStack, cm := stack.Node.ComputeNode, stack.IpfsStack, stack.Node.CleanupManager
		defer cm.Cleanup()

		cid, _ := devstack.AddTextToNodes(ctx, []byte(text), ipfsStack.IPFSClients[0])

		result, _, err := computeNode.SelectJob(ctx, computenode.JobSelectionPolicyProbeData{
			NodeID: "test",
			JobID:  "test",
			Spec: model.Spec{
				Engine:   model.EngineDocker,
				Verifier: model.VerifierNoop,
				Resources: model.ResourceUsageConfig{
					CPU:    "100m",
					Memory: "100mb",
					// we simulate having calculated the disk size here
					Disk: "6b",
				},
				Inputs: []model.StorageSpec{
					{
						StorageSource: model.StorageSourceIPFS,
						CID:           cid,
						Path:          "/data/file.txt",
					},
				},
				Docker: model.JobSpecDocker{
					Image: "ubuntu",
					Entrypoint: []string{
						"bash",
						"-c",
						"/data/file.txt",
					},
				},
			},
		})

		require.NoError(suite.T(), err)
		require.Equal(suite.T(), expected, result)
	}

	runTest("hello from 1b test", "1b", false)
	runTest("hello from 1k test", "1k", true)

}

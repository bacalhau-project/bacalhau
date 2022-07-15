package computenode

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/capacitymanager"
	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/controller"
	"github.com/filecoin-project/bacalhau/pkg/datastore/inmemory"
	devstack "github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	executor_util "github.com/filecoin-project/bacalhau/pkg/executor/util"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/inprocess"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	verifier_util "github.com/filecoin-project/bacalhau/pkg/verifier/util"
	"github.com/stretchr/testify/require"
)

func TestDockerResourceLimitsCPU(t *testing.T) {

	CPU_LIMIT := "100m"

	computeNode, _, cm := SetupTestDockerIpfs(t, computenode.NewDefaultComputeNodeConfig())
	defer cm.Cleanup()

	// this will give us a numerator and denominator that should end up at the
	// same 0.1 value that 100m means
	// https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/8/html/managing_monitoring_and_updating_the_kernel/using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications_managing-monitoring-and-updating-the-kernel#proc_controlling-distribution-of-cpu-time-for-applications-by-adjusting-cpu-bandwidth_using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications
	result := RunJobGetStdout(t, computeNode, executor.JobSpec{
		Engine:   executor.EngineDocker,
		Verifier: verifier.VerifierNoop,
		Resources: capacitymanager.ResourceUsageConfig{
			CPU:    CPU_LIMIT,
			Memory: "100mb",
		},
		Docker: executor.JobSpecDocker{
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
	require.NoError(t, err)

	denominator, err := strconv.Atoi(values[1])
	require.NoError(t, err)

	var containerCPU float64 = 0

	if denominator > 0 {
		containerCPU = float64(numerator) / float64(denominator)
	}

	require.Equal(t, capacitymanager.ConvertCPUString(CPU_LIMIT), containerCPU, "the container reported CPU does not equal the configured limit")
}

func TestDockerResourceLimitsMemory(t *testing.T) {

	MEMORY_LIMIT := "100mb"

	computeNode, _, cm := SetupTestDockerIpfs(t, computenode.NewDefaultComputeNodeConfig())
	defer cm.Cleanup()

	result := RunJobGetStdout(t, computeNode, executor.JobSpec{
		Engine:   executor.EngineDocker,
		Verifier: verifier.VerifierNoop,
		Resources: capacitymanager.ResourceUsageConfig{
			CPU:    "100m",
			Memory: MEMORY_LIMIT,
		},
		Docker: executor.JobSpecDocker{
			Image: "ubuntu",
			Entrypoint: []string{
				"bash",
				"-c",
				"cat /sys/fs/cgroup/memory.max",
			},
		},
	})

	intVar, err := strconv.Atoi(strings.TrimSpace(result))
	require.NoError(t, err)
	require.Equal(t, capacitymanager.ConvertMemoryString(MEMORY_LIMIT), uint64(intVar), "the container reported memory does not equal the configured limit")
}

func TestDockerResourceLimitsDisk(t *testing.T) {

	runTest := func(text, diskSize string, expected bool) {
		computeNode, ipfsStack, cm := SetupTestDockerIpfs(t, computenode.ComputeNodeConfig{
			CapacityManagerConfig: capacitymanager.Config{
				ResourceLimitTotal: capacitymanager.ResourceUsageConfig{
					// so we have a compute node with 1 byte of disk space
					Disk: diskSize,
				},
			},
		})
		defer cm.Cleanup()

		cid, err := ipfsStack.AddTextToNodes(1, []byte(text))

		result, _, err := computeNode.SelectJob(context.Background(), computenode.JobSelectionPolicyProbeData{
			NodeID: "test",
			JobID:  "test",
			Spec: executor.JobSpec{
				Engine:   executor.EngineDocker,
				Verifier: verifier.VerifierNoop,
				Resources: capacitymanager.ResourceUsageConfig{
					CPU:    "100m",
					Memory: "100mb",
					// we simulate having calculated the disk size here
					Disk: "6b",
				},
				Inputs: []storage.StorageSpec{
					{
						Engine: storage.IPFSDefault,
						Cid:    cid,
						Path:   "/data/file.txt",
					},
				},
				Docker: executor.JobSpecDocker{
					Image: "ubuntu",
					Entrypoint: []string{
						"bash",
						"-c",
						"/data/file.txt",
					},
				},
			},
		})

		require.NoError(t, err)
		require.Equal(t, expected, result)
	}

	runTest("hello", "1b", false)
	runTest("hello", "1k", true)

}

// how many bytes more does ipfs report the file than the actual content?
const IpfsMetadataSize = 8

func TestGetVolumeSize(t *testing.T) {

	runTest := func(text string, expected uint64) {

		cm := system.NewCleanupManager()

		ipfsStack, err := devstack.NewDevStackIPFS(cm, 1)
		require.NoError(t, err)

		apiAddress := ipfsStack.Nodes[0].IpfsClient.APIAddress()
		transport, err := inprocess.NewInprocessTransport()
		require.NoError(t, err)

		datastore, err := inmemory.NewInMemoryDatastore()
		require.NoError(t, err)

		ctrl, err := controller.NewController(cm, datastore, transport)
		require.NoError(t, err)

		executors, err := executor_util.NewStandardExecutors(cm, apiAddress, "devstacknode0")
		require.NoError(t, err)

		verifiers, err := verifier_util.NewIPFSVerifiers(cm, apiAddress)
		require.NoError(t, err)

		_, err = computenode.NewComputeNode(
			cm,
			ctrl,
			executors,
			verifiers,
			computenode.ComputeNodeConfig{},
		)
		require.NoError(t, err)

		cid, err := ipfsStack.AddTextToNodes(1, []byte(text))
		require.NoError(t, err)

		executor := executors[executor.EngineDocker]

		result, err := executor.GetVolumeSize(context.Background(), storage.StorageSpec{
			Engine: storage.IPFSDefault,
			Cid:    cid,
			Path:   "/",
		})

		require.NoError(t, err)
		require.Equal(t, expected+IpfsMetadataSize, result)
	}

	runTest("hello", 5)
	runTest("hello world", 11)

}

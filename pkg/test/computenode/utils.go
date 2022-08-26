package computenode

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/capacitymanager"
	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/controller"
	devstack "github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	executor_util "github.com/filecoin-project/bacalhau/pkg/executor/util"
	"github.com/filecoin-project/bacalhau/pkg/localdb/inmemory"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	publisher_util "github.com/filecoin-project/bacalhau/pkg/publisher/util"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	noop_storage "github.com/filecoin-project/bacalhau/pkg/storage/noop"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/inprocess"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	verifier_util "github.com/filecoin-project/bacalhau/pkg/verifier/util"
	"github.com/stretchr/testify/require"
)

// setup a docker ipfs stack to run compute node tests against
func SetupTestDockerIpfs(
	t *testing.T,
	config computenode.ComputeNodeConfig, //nolint:gocritic
) (*computenode.ComputeNode, *devstack.DevStackIPFS, *system.CleanupManager) {
	cm := system.NewCleanupManager()

	ipfsStack, err := devstack.NewDevStackIPFS(cm, 1)
	require.NoError(t, err)

	apiAddress := ipfsStack.Nodes[0].IpfsClient.APIAddress()
	transport, err := inprocess.NewInprocessTransport()
	require.NoError(t, err)

	datastore, err := inmemory.NewInMemoryDatastore()
	require.NoError(t, err)

	ipfsID := ipfsStack.Nodes[0].IpfsNode.ID()

	storageProviders, err := executor_util.NewStandardStorageProviders(cm, executor_util.StandardStorageProviderOptions{
		IPFSMultiaddress: apiAddress,
	})
	require.NoError(t, err)
	executors, err := executor_util.NewStandardExecutors(
		cm,
		executor_util.StandardExecutorOptions{
			DockerID: fmt.Sprintf("devstacknode0-%s", ipfsID),
			Storage: executor_util.StandardStorageProviderOptions{
				IPFSMultiaddress: apiAddress,
			},
		},
	)
	require.NoError(t, err)

	ctrl, err := controller.NewController(cm, datastore, transport, storageProviders)
	require.NoError(t, err)

	verifiers, err := verifier_util.NewNoopVerifiers(
		cm,
		ctrl.GetStateResolver(),
	)
	require.NoError(t, err)

	publishers, err := publisher_util.NewIPFSPublishers(
		cm,
		ctrl.GetStateResolver(),
		apiAddress,
	)
	require.NoError(t, err)

	computeNode, err := computenode.NewComputeNode(
		cm,
		ctrl,
		executors,
		verifiers,
		publishers,
		config,
	)
	require.NoError(t, err)

	return computeNode, ipfsStack, cm
}

func SetupTestNoop(
	t *testing.T,
	//nolint:gocritic
	computeNodeconfig computenode.ComputeNodeConfig,
	noopExecutorConfig noop_executor.ExecutorConfig,
) (*computenode.ComputeNode, *requesternode.RequesterNode, *controller.Controller, *system.CleanupManager) {
	cm := system.NewCleanupManager()

	transport, err := inprocess.NewInprocessTransport()
	require.NoError(t, err)

	datastore, err := inmemory.NewInMemoryDatastore()
	require.NoError(t, err)

	executors, err := executor_util.NewNoopExecutors(cm, noopExecutorConfig)
	require.NoError(t, err)

	storageProviders, err := executor_util.NewNoopStorageProviders(cm, noop_storage.StorageConfig{})
	require.NoError(t, err)

	ctrl, err := controller.NewController(cm, datastore, transport, storageProviders)
	require.NoError(t, err)

	verifiers, err := verifier_util.NewNoopVerifiers(cm, ctrl.GetStateResolver())
	require.NoError(t, err)

	publishers, err := publisher_util.NewNoopPublishers(cm, ctrl.GetStateResolver())
	require.NoError(t, err)

	requestorNode, err := requesternode.NewRequesterNode(
		cm,
		ctrl,
		verifiers,
		requesternode.RequesterNodeConfig{},
	)
	require.NoError(t, err)

	computeNode, err := computenode.NewComputeNode(
		cm,
		ctrl,
		executors,
		verifiers,
		publishers,
		computeNodeconfig,
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	err = ctrl.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if err = transport.Start(ctx); err != nil {
		t.Fatal(err)
	}

	return computeNode, requestorNode, ctrl, cm
}

func GetJobSpec(cid string) executor.JobSpec {
	inputs := []storage.StorageSpec{}
	if cid != "" {
		inputs = []storage.StorageSpec{
			{
				Engine: storage.StorageSourceIPFS,
				Cid:    cid,
				Path:   "/test_file.txt",
			},
		}
	}
	return executor.JobSpec{
		Engine:   executor.EngineDocker,
		Verifier: verifier.VerifierNoop,
		Docker: executor.JobSpecDocker{
			Image: "ubuntu",
			Entrypoint: []string{
				"cat",
				"/test_file.txt",
			},
		},
		Inputs: inputs,
	}
}

func GetProbeData(cid string) computenode.JobSelectionPolicyProbeData {
	return computenode.JobSelectionPolicyProbeData{
		NodeID: "test",
		JobID:  "test",
		Spec:   GetJobSpec(cid),
	}
}

//nolint:unused,deadcode
func getResources(c, m, d string) capacitymanager.ResourceUsageConfig {
	return capacitymanager.ResourceUsageConfig{
		CPU:    c,
		Memory: m,
		Disk:   d,
	}
}

//nolint:unused,deadcode
func getResourcesArray(data [][]string) []capacitymanager.ResourceUsageConfig {
	var res []capacitymanager.ResourceUsageConfig
	for _, d := range data {
		res = append(res, getResources(d[0], d[1], d[2]))
	}
	return res
}

func RunJobGetStdout(
	t *testing.T,
	computeNode *computenode.ComputeNode,
	spec executor.JobSpec,
) string {
	result, err := ioutil.TempDir("", "bacalhau-RunJobGetStdout")
	require.NoError(t, err)
	err = computeNode.RunShardExecution(context.Background(), executor.Job{
		ID:   "test",
		Spec: spec,
	}, 0, result)
	require.NoError(t, err)

	stdoutPath := fmt.Sprintf("%s/stdout", result)
	require.DirExists(t, result, "The job result folder exists")
	require.FileExists(t, stdoutPath, "The stdout file exists")
	dat, err := os.ReadFile(stdoutPath)
	require.NoError(t, err)
	return string(dat)
}

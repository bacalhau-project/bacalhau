package computenode

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/stretchr/testify/require"
)

// Setup a docker ipfs devstack to run compute node tests against
func SetupTestDockerIpfs(
	t *testing.T,
	config computenode.ComputeNodeConfig, //nolint:gocritic
) (*computenode.ComputeNode, *devstack.DevStackIPFS, *system.CleanupManager) {
	// TODO @enricorotundo #493: needed here?
	// system.InitConfigForTesting(t)
	
	cm := system.NewCleanupManager()

	ipfsStack, err := devstack.NewDevStackIPFS(cm, 1)
	require.NoError(t, err)

	apiAddress := ipfsStack.Nodes[0].IpfsClient.APIAddress()
	transport, err := inprocess.NewInprocessTransport()
	require.NoError(t, err)

	datastore, err := inmemory.NewInMemoryDatastore()
	require.NoError(t, err)

	ipfsID := ipfsStack.Nodes[0].IpfsNode.ID()

	storageProviders, err := executor_util.NewStandardStorageProviders(cm, apiAddress)
	require.NoError(t, err)
	executors, err := executor_util.NewStandardExecutors(
		cm,
		apiAddress,
		fmt.Sprintf("devstacknode0-%s", ipfsID),
	)
	require.NoError(t, err)

	verifiers, err := verifier_util.NewIPFSVerifiers(
		cm,
		apiAddress,
		job.NewNoopJobLoader(),
		job.NewNoopStateLoader(),
	)
	require.NoError(t, err)

	ctrl, err := controller.NewController(cm, datastore, transport, storageProviders)
	require.NoError(t, err)

	computeNode, err := computenode.NewComputeNode(
		cm,
		ctrl,
		executors,
		verifiers,
		config,
	)
	require.NoError(t, err)

	return computeNode, ipfsStack, cm
}

// TODO @enricorotundo #493: move this to dedicated test tooling package?
// TODO @enricorotundo #493: REFACTOR - align SetupTestNoop with SetupTestDockerIpfs
// setup a full noop stack to run tests against
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

	verifiers, err := verifier_util.NewNoopVerifiers(cm)
	require.NoError(t, err)

	storageProviders, err := executor_util.NewNoopStorageProviders(cm)
	require.NoError(t, err)

	ctrl, err := controller.NewController(cm, datastore, transport, storageProviders)
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
		inputs = []model.StorageSpec{
			{
				Engine: model.StorageSourceIPFS,
				Cid:    cid,
				Path:   "/test_file.txt",
			},
		}
	}
	return model.JobSpec{
		Engine:   model.EngineDocker,
		Verifier: model.VerifierNoop,
		Docker: model.JobSpecDocker{
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
func getResources(c, m, d string) model.ResourceUsageConfig {
	return model.ResourceUsageConfig{
		CPU:    c,
		Memory: m,
		Disk:   d,
	}
}

//nolint:unused,deadcode
func getResourcesArray(data [][]string) []model.ResourceUsageConfig {
	var res []model.ResourceUsageConfig
	for _, d := range data {
		res = append(res, getResources(d[0], d[1], d[2]))
	}
	return res
}

func RunJobGetStdout(
	t *testing.T,
	ctx context.Context,
	computeNode *computenode.ComputeNode,
	spec model.JobSpec,
) string {
	result, err := ioutil.TempDir("", "bacalhau-RunJobGetStdout")
	require.NoError(t, err)

	job := model.Job{
		ID:   "test",
		Spec: spec,
	}
	shard := model.JobShard{
		Job:   job,
		Index: 0,
	}
	err = computeNode.RunShardExecution(ctx, shard, result)
	require.NoError(t, err)

	stdoutPath := fmt.Sprintf("%s/stdout", result)
	require.DirExists(t, result, "The job result folder exists")
	require.FileExists(t, stdoutPath, "The stdout file exists")
	dat, err := os.ReadFile(stdoutPath)
	require.NoError(t, err)
	return string(dat)
}

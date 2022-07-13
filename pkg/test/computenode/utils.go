package computenode

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/filecoin-project/bacalhau/pkg/computenode"
	devstack "github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	executor_util "github.com/filecoin-project/bacalhau/pkg/executor/util"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/requestornode"
	"github.com/filecoin-project/bacalhau/pkg/resourceusage"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/inprocess"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	verifier_util "github.com/filecoin-project/bacalhau/pkg/verifier/util"
	"github.com/stretchr/testify/assert"
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

	executors, err := executor_util.NewStandardExecutors(
		cm, apiAddress, "devstacknode0")
	require.NoError(t, err)

	verifiers, err := verifier_util.NewIPFSVerifiers(cm, apiAddress)
	require.NoError(t, err)

	computeNode, err := computenode.NewComputeNode(
		cm,
		transport,
		executors,
		verifiers,
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
) (*computenode.ComputeNode, *requestornode.RequesterNode, *system.CleanupManager) {
	cm := system.NewCleanupManager()

	transport, err := inprocess.NewInprocessTransport()
	require.NoError(t, err)

	executors, err := executor_util.NewNoopExecutors(cm, noopExecutorConfig)
	require.NoError(t, err)

	verifiers, err := verifier_util.NewNoopVerifiers(cm)
	require.NoError(t, err)

	requestorNode, err := requestornode.NewRequesterNode(
		cm,
		transport,
		verifiers,
	)
	require.NoError(t, err)

	computeNode, err := computenode.NewComputeNode(
		cm,
		transport,
		executors,
		verifiers,
		computeNodeconfig,
	)
	if err != nil {
		spew.Dump(computeNodeconfig)
		t.Fatal(err)
	}

	return computeNode, requestorNode, cm
}

func GetJobSpec(cid string) executor.JobSpec {
	inputs := []storage.StorageSpec{}
	if cid != "" {
		inputs = []storage.StorageSpec{
			{
				Engine: "ipfs",
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
func getResources(c, m, d string) resourceusage.ResourceUsageConfig {
	return resourceusage.ResourceUsageConfig{
		CPU:    c,
		Memory: m,
		Disk:   d,
	}
}

//nolint:unused,deadcode
func getResourcesArray(data [][]string) []resourceusage.ResourceUsageConfig {
	var res []resourceusage.ResourceUsageConfig
	for _, d := range data {
		res = append(res, getResources(d[0], d[1], d[2]))
	}
	return res
}

// given a transport interface - run a job from start to end
// basically acting as an "auto requestor" node
// that will submit the job and then accept any bids
// that come in (up until the concurrency)
func RunJobViaRequestor(
	requestor requestornode.RequesterNode,
	job *executor.JobSpec,
) error {
	return nil
}

func RunJobGetStdout(
	t *testing.T,
	computeNode *computenode.ComputeNode,
	spec executor.JobSpec,
) string {
	result, err := computeNode.RunJob(context.Background(), executor.Job{
		ID:   "test",
		Spec: spec,
	})
	assert.NoError(t, err)

	stdoutPath := fmt.Sprintf("%s/stdout", result)
	assert.DirExists(t, result, "The job result folder exists")
	assert.FileExists(t, stdoutPath, "The stdout file exists")
	dat, err := os.ReadFile(stdoutPath)
	assert.NoError(t, err)
	return string(dat)
}

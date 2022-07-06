package computenode

import (
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
)

// setup a docker ipfs stack to run compute node tests against
func SetupTestDockerIpfs(
	t *testing.T,
	config computenode.ComputeNodeConfig, //nolint:gocritic
) (*computenode.ComputeNode, *devstack.DevStackIPFS, *system.CleanupManager) {
	cm := system.NewCleanupManager()

	ipfsStack, err := devstack.NewDevStackIPFS(cm, 1)
	if err != nil {
		t.Fatal(err)
	}

	apiAddress := ipfsStack.Nodes[0].IpfsClient.APIAddress()
	transport, err := inprocess.NewInprocessTransport()
	if err != nil {
		t.Fatal(err)
	}

	executors, err := executor_util.NewStandardExecutors(
		cm, apiAddress, "devstacknode0")
	if err != nil {
		t.Fatal(err)
	}

	verifiers, err := verifier_util.NewIPFSVerifiers(cm, apiAddress)
	if err != nil {
		t.Fatal(err)
	}

	computeNode, err := computenode.NewComputeNode(
		cm,
		transport,
		executors,
		verifiers,
		config,
	)
	if err != nil {
		t.Fatal(err)
	}

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
	if err != nil {
		t.Fatal(err)
	}

	executors, err := executor_util.NewNoopExecutors(cm, noopExecutorConfig)
	if err != nil {
		t.Fatal(err)
	}

	verifiers, err := verifier_util.NewNoopVerifiers(cm)
	if err != nil {
		t.Fatal(err)
	}

	requestorNode, err := requestornode.NewRequesterNode(
		cm,
		transport,
		verifiers,
	)
	if err != nil {
		t.Fatal(err)
	}

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

func GetJobSpec(cid string) *executor.JobSpec {
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
	return &executor.JobSpec{
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
func getResources(c, m string) resourceusage.ResourceUsageConfig {
	return resourceusage.ResourceUsageConfig{
		CPU:    c,
		Memory: m,
	}
}

//nolint:unused,deadcode
func getResourcesArray(data [][]string) []resourceusage.ResourceUsageConfig {
	var res []resourceusage.ResourceUsageConfig
	for _, d := range data {
		res = append(res, getResources(d[0], d[1]))
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

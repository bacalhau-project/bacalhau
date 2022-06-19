package compute_node

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/compute_node"
	devstack "github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	executor_util "github.com/filecoin-project/bacalhau/pkg/executor/util"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/inprocess"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	verifier_util "github.com/filecoin-project/bacalhau/pkg/verifier/util"
)

func SetupTest(
	t *testing.T,
	jobSelectionPolicy compute_node.JobSelectionPolicy,
) (*compute_node.ComputeNode, *devstack.DevStack_IPFS, *system.CleanupManager) {

	cm := system.NewCleanupManager()

	ipfsStack, err := devstack.NewDevStack_IPFS(cm, 1)
	if err != nil {
		t.Fatal(err)
	}

	apiAddress := ipfsStack.Nodes[0].IpfsNode.ApiAddress()
	transport, err := inprocess.NewInprocessTransport()
	if err != nil {
		t.Fatal(err)
	}

	executors, err := executor_util.NewDockerIPFSExecutors(
		cm, apiAddress, "devstacknode0")
	if err != nil {
		t.Fatal(err)
	}

	verifiers, err := verifier_util.NewIPFSVerifiers(cm, apiAddress)
	if err != nil {
		t.Fatal(err)
	}

	computeNode, err := compute_node.NewComputeNode(
		transport,
		executors,
		verifiers,
		jobSelectionPolicy,
	)
	if err != nil {
		t.Fatal(err)
	}

	return computeNode, ipfsStack, cm
}

func GetJobSpec(cid string) *types.JobSpec {
	inputs := []types.StorageSpec{}
	if cid != "" {
		inputs = []types.StorageSpec{
			{
				Engine: "ipfs",
				Cid:    cid,
				Path:   "/test_file.txt",
			},
		}
	}
	return &types.JobSpec{
		Engine:   string(executor.EXECUTOR_DOCKER),
		Verifier: string(verifier.VERIFIER_NOOP),
		VM: types.JobSpecVm{
			Image: "ubuntu",
			Entrypoint: []string{
				"cat",
				"/test_file.txt",
			},
		},
		Inputs: inputs,
	}
}

func GetProbeData(cid string) compute_node.JobSelectionPolicyProbeData {
	return compute_node.JobSelectionPolicyProbeData{
		NodeId: "test",
		JobId:  "test",
		Spec:   GetJobSpec(cid),
	}
}

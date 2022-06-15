package devstack

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/compute_node"
	devstack "github.com/filecoin-project/bacalhau/pkg/devstack"
	executor_util "github.com/filecoin-project/bacalhau/pkg/executor/util"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/inprocess"
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

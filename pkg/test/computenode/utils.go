package computenode

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	devstack "github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	executor_util "github.com/filecoin-project/bacalhau/pkg/executor/util"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/inprocess"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	verifier_util "github.com/filecoin-project/bacalhau/pkg/verifier/util"
)

func SetupTest(
	t *testing.T,
	jobSelectionPolicy computenode.JobSelectionPolicy,
) (*computenode.ComputeNode, *devstack.DevStackIPFS, *system.CleanupManager) {
	cm := system.NewCleanupManager()

	ipfsStack, err := devstack.NewDevStackIPFS(cm, 1)
	if err != nil {
		t.Fatal(err)
	}

	apiAddress := ipfsStack.Nodes[0].IpfsNode.APIAddress()
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

	computeNode, err := computenode.NewComputeNode(
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
		VM: executor.JobSpecVM{
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

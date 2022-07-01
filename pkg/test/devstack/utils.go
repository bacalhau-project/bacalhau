package devstack

import (
	"fmt"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	executor_util "github.com/filecoin-project/bacalhau/pkg/executor/util"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	verifier_util "github.com/filecoin-project/bacalhau/pkg/verifier/util"
	"github.com/stretchr/testify/assert"
)

var StorageDriverNames = []string{
	storage.IPFSFuseDocker,
	storage.IPFSAPICopy,
}

func SetupTest(
	t *testing.T,
	nodes int, badActors int,
	jobSelectionPolicy computenode.JobSelectionPolicy,
) (*devstack.DevStack, *system.CleanupManager) {
	system.InitConfigForTesting(t)

	cm := system.NewCleanupManager()
	getExecutors := func(ipfsMultiAddress string, nodeIndex int) (
		map[executor.EngineType]executor.Executor, error) {
		return executor_util.NewStandardExecutors(
			cm, ipfsMultiAddress, fmt.Sprintf("devstacknode%d", nodeIndex))
	}
	getVerifiers := func(ipfsMultiAddress string, nodeIndex int) (
		map[verifier.VerifierType]verifier.Verifier, error) {
		return verifier_util.NewIPFSVerifiers(cm, ipfsMultiAddress)
	}
	stack, err := devstack.NewDevStack(
		cm,
		nodes,
		badActors,
		getExecutors,
		getVerifiers,
		jobSelectionPolicy,
	)
	assert.NoError(t, err)

	// important to give the pubsub network time to connect
	time.Sleep(time.Second)

	return stack, cm
}

func TeardownTest(stack *devstack.DevStack, cm *system.CleanupManager) {
	stack.PrintNodeInfo()
	cm.Cleanup()
}

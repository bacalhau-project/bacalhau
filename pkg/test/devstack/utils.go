package devstack

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/controller"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	executor_util "github.com/filecoin-project/bacalhau/pkg/executor/util"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	verifier_util "github.com/filecoin-project/bacalhau/pkg/verifier/util"
	"github.com/stretchr/testify/require"
)

var StorageNames = []storage.StorageSourceType{
	storage.StorageSourceIPFS,
}

func SetupTest(
	t *testing.T,
	nodes int, badActors int,
	//nolint:gocritic
	config computenode.ComputeNodeConfig,
) (*devstack.DevStack, *system.CleanupManager) {
	// TODO @enricorotundo #493: needed every time?
	system.InitConfigForTesting(t)

	cm := system.NewCleanupManager()
	getStorageProviders := func(ipfsMultiAddress string, nodeIndex int) (map[storage.StorageSourceType]storage.StorageProvider, error) {
		return executor_util.NewStandardStorageProviders(cm, ipfsMultiAddress)
	}
	getExecutors := func(
		ipfsMultiAddress string,
		nodeIndex int,
		ctrl *controller.Controller,
	) (
		map[executor.EngineType]executor.Executor,
		error,
	) {
		ipfsParts := strings.Split(ipfsMultiAddress, "/")
		ipfsSuffix := ipfsParts[len(ipfsParts)-1]
		return executor_util.NewStandardExecutors(
			cm,
			ipfsMultiAddress,
			fmt.Sprintf("devstacknode%d-%s", nodeIndex, ipfsSuffix),
		)
	}
	getVerifiers := func(
		ipfsMultiAddress string,
		nodeIndex int,
		ctrl *controller.Controller,
	) (
		map[verifier.VerifierType]verifier.Verifier,
		error,
	) {
		jobLoader := func(ctx context.Context, id string) (executor.Job, error) {
			return ctrl.GetJob(ctx, id)
		}
		stateLoader := func(ctx context.Context, id string) (executor.JobState, error) {
			return ctrl.GetJobState(ctx, id)
		}
		return verifier_util.NewIPFSVerifiers(cm, ipfsMultiAddress, jobLoader, stateLoader)
	}
	stack, err := devstack.NewDevStack(
		cm,
		nodes,
		badActors,
		getStorageProviders,
		getExecutors,
		getVerifiers,
		config,
		"",
		false,
	)
	require.NoError(t, err)

	// important to give the pubsub network time to connect
	time.Sleep(time.Second)

	return stack, cm
}

func TeardownTest(stack *devstack.DevStack, cm *system.CleanupManager) {
	stack.PrintNodeInfo()
	cm.Cleanup()
}

package devstack

import (
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
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	publisher_util "github.com/filecoin-project/bacalhau/pkg/publisher/util"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/libp2p"
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
	system.InitConfigForTesting(t)

	cm := system.NewCleanupManager()
	getStorageProviders := func(ipfsMultiAddress string, nodeIndex int) (map[storage.StorageSourceType]storage.StorageProvider, error) {
		return executor_util.NewStandardStorageProviders(cm, executor_util.StandardStorageProviderOptions{
			IPFSMultiaddress: ipfsMultiAddress,
		})
	}
	getExecutors := func(
		ipfsMultiAddress string,
		nodeIndex int,
		isBadActor bool,
		ctrl *controller.Controller,
	) (
		map[executor.EngineType]executor.Executor,
		error,
	) {
		ipfsParts := strings.Split(ipfsMultiAddress, "/")
		ipfsSuffix := ipfsParts[len(ipfsParts)-1]
		return executor_util.NewStandardExecutors(
			cm,
			executor_util.StandardExecutorOptions{
				DockerID:   fmt.Sprintf("devstacknode%d-%s", nodeIndex, ipfsSuffix),
				IsBadActor: isBadActor,
				Storage: executor_util.StandardStorageProviderOptions{
					IPFSMultiaddress: ipfsMultiAddress,
				},
			},
		)
	}
	getVerifiers := func(
		transport *libp2p.LibP2PTransport,
		nodeIndex int,
		ctrl *controller.Controller,
	) (
		map[verifier.VerifierType]verifier.Verifier,
		error,
	) {
		return verifier_util.NewStandardVerifiers(
			cm,
			ctrl.GetStateResolver(),
			transport.Encrypt,
			transport.Decrypt,
		)
	}
	getPublishers := func(
		ipfsMultiAddress string,
		nodeIndex int,
		ctrl *controller.Controller,
	) (
		map[publisher.PublisherType]publisher.Publisher,
		error,
	) {
		return publisher_util.NewIPFSPublishers(cm, ctrl.GetStateResolver(), ipfsMultiAddress)
	}
	stack, err := devstack.NewDevStack(
		cm,
		nodes,
		badActors,
		getStorageProviders,
		getExecutors,
		getVerifiers,
		getPublishers,
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

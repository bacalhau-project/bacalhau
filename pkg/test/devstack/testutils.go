package devstack

import (
	"fmt"
	"os"
	"testing"

	"context"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/assert"

	"github.com/rs/zerolog/log"
)

func setupTest(
	t *testing.T,
	nodes int,
	badActors int,
) (*devstack.DevStack, context.CancelFunc) {
	ctx, cancelFunction := system.GetCancelContext()

	getExecutors := func(ipfsMultiAddress string) (map[string]executor.Executor, error) {
		return devstack.NewDockerIPFSExecutors(ctx, ipfsMultiAddress)
	}

	stack, err := devstack.NewDevStack(
		ctx,
		nodes,
		badActors,
		getExecutors,
	)
	assert.NoError(t, err)
	if err != nil {
		log.Fatal().Msg(fmt.Sprintf("Unable to create devstack: %s", err))
	}
	// TODO: add a waitgroup with checks on each part of a node
	// (i.e. libp2p connected, jsonrpc serving, ipfs functional)
	time.Sleep(time.Second * 2)
	return stack, cancelFunction
}

// this might be called multiple times if KEEP_STACK is active
// the first time - once the test has completed, this function will be called
// it will reset the KEEP_STACK variable so the user can ctrl+c the running stack
func teardownTest(stack *devstack.DevStack, cancelFunction context.CancelFunc) {
	if os.Getenv("KEEP_STACK") == "" {
		cancelFunction()
		// need some time to let ipfs processes shut down
		time.Sleep(time.Second * 2)
	} else {
		stack.PrintNodeInfo()
		os.Setenv("KEEP_STACK", "")
		select {}
	}
}

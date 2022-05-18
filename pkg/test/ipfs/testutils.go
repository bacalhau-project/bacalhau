package ipfs

import (
	"fmt"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/assert"

	"github.com/rs/zerolog/log"
)

func SetupTest(
	t *testing.T,
	nodes int,
) (*devstack.DevStack_IPFS, *system.CancelContext) {
	cancelContext := system.GetCancelContextWithSignals()
	stack, err := devstack.NewDevStack_IPFS(
		cancelContext,
		nodes,
	)
	assert.NoError(t, err)
	if err != nil {
		log.Fatal().Msg(fmt.Sprintf("Unable to create devstack: %s", err))
	}
	return stack, cancelContext
}

func TeardownTest(stack *devstack.DevStack_IPFS, cancelContext *system.CancelContext) {
	cancelContext.Stop()
	if system.ShouldKeepStack() {
		stack.PrintNodeInfo()
		select {}
	}
}

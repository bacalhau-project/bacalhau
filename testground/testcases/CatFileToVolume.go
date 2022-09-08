package testcases

import (
	"context"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	"github.com/filecoin-project/bacalhau/testground/utils"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"time"
)

func CatFileToVolume(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return utils.ExecuteTest(ctx, runenv, initCtx, executeCatFileToStdout)
}

func executeCatFileToVolume(ctx context.Context, runenv *runtime.RunEnv, initCtx *run.InitContext, node *node.Node) error {
	return utils.RunDockerTest(runenv, ctx, scenario.CatFileToVolume(), node, 3)
}

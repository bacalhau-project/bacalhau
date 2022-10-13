//go:build !(unit && (windows || darwin))

package devstack

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLotusNode(t *testing.T) {
	require.NoError(t, system.InitConfigForTesting())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	stack, _ := SetupTest(ctx, t, 1, 0, true, computenode.NewDefaultComputeNodeConfig())

	require.NotNil(t, stack.Lotus)
	assert.NotEmpty(t, stack.Lotus.Dir)
	require.NotEmpty(t, stack.Lotus.Token)
	require.NotEmpty(t, stack.Lotus.Port)

	lotus := lotusApi(t, ctx, stack.Lotus.Port, stack.Lotus.Token)

	version, err := lotus.Version(ctx)
	require.NoError(t, err)

	t.Log(version.Version)
}

func lotusApi(t *testing.T, ctx context.Context, port string, token string) *lotusNodeCommonStruct {
	headers := http.Header{"Authorization": []string{fmt.Sprintf("Bearer %s", token)}}
	addr := fmt.Sprintf("ws://localhost:%s/rpc/v0", port)

	var lotus lotusNodeCommonStruct

	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin", []interface{}{&lotus.Internal}, headers)
	require.NoError(t, err)
	t.Cleanup(func() {
		closer()
	})

	return &lotus
}

// Importing the Lotus API currently causes dependency issues, so only including the smallest part needed
type lotusNodeCommonStruct struct {
	Internal struct {
		Version func(p0 context.Context) (APIVersion, error) `perm:"read"`
	}
}

func (l *lotusNodeCommonStruct) Version(ctx context.Context) (APIVersion, error) {
	return l.Internal.Version(ctx)
}

type APIVersion struct {
	Version    string
	APIVersion Version
	BlockDelay uint64
}

type Version uint32

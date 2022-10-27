//go:build !(unit && (windows || darwin))

package devstack

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/multiformats/go-multiaddr"
	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type LotusNodeSuite struct {
	suite.Suite
}

func TestLotusNodeSuite(t *testing.T) {
	suite.Run(t, new(LotusNodeSuite))
}

func (s *LotusNodeSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	require.NoError(s.T(), system.InitConfigForTesting())
}

func (s *LotusNodeSuite) TearDownTest() {}

func (s *LotusNodeSuite) TestLotusNode() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	stack, _ := SetupTest(ctx, s.T(), 1, 0, true, computenode.NewDefaultComputeNodeConfig())

	require.NotNil(s.T(), stack.Lotus)
	assert.DirExists(s.T(), stack.Lotus.UploadDir)
	require.DirExists(s.T(), stack.Lotus.PathDir)
	require.FileExists(s.T(), filepath.Join(stack.Lotus.PathDir, "config.toml"))
	require.FileExists(s.T(), filepath.Join(stack.Lotus.PathDir, "token"))

	token, err := os.ReadFile(filepath.Join(stack.Lotus.PathDir, "token"))
	require.NoError(s.T(), err)

	configFile, err := os.ReadFile(filepath.Join(stack.Lotus.PathDir, "config.toml"))
	require.NoError(s.T(), err)

	var config struct {
		API struct {
			ListenAddress string
		}
	}
	require.NoError(s.T(), toml.Unmarshal(configFile, &config))

	multiAddr, err := multiaddr.NewMultiaddr(config.API.ListenAddress)
	require.NoError(s.T(), err)

	com, addr := multiaddr.SplitFirst(multiAddr)
	assert.Equal(s.T(), "ip4", com.Protocol().Name)
	assert.Equal(s.T(), "0.0.0.0", com.Value())
	port, addr := multiaddr.SplitFirst(addr)
	assert.Equal(s.T(), "tcp", port.Protocol().Name)
	assert.NotEmpty(s.T(), port.Value())
	com, addr = multiaddr.SplitFirst(addr)
	assert.Equal(s.T(), "http", com.Protocol().Name)

	lotus := lotusApi(s.T(), ctx, port.Value(), string(token))

	version, err := lotus.Version(ctx)
	require.NoError(s.T(), err)

	s.T().Log(version.Version)
}

func lotusApi(t *testing.T, ctx context.Context, port string, token string) *lotusNodeCommonStruct {
	headers := http.Header{"Authorization": []string{fmt.Sprintf("Bearer %s", token)}}
	addr := fmt.Sprintf("ws://localhost:%s/rpc/v1", port)

	var lotus lotusNodeCommonStruct

	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin", []interface{}{&lotus.Internal}, headers)
	require.NoError(t, err)
	t.Cleanup(closer)

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

//go:build unit || !integration

package compute_test

import (
	"context"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/lib/backoff"
	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	natsutil "github.com/bacalhau-project/bacalhau/pkg/nats"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol"
	nclprotocolcompute "github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol/compute"
	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol/dispatcher"
	ncltest "github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol/test"
)

type ConfigTestSuite struct {
	suite.Suite
	ctrl             *gomock.Controller
	nodeInfoProvider *ncltest.MockNodeInfoProvider
	messageHandler   *ncl.MockMessageHandler
	checkpointer     *nclprotocol.MockCheckpointer
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

func (s *ConfigTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.nodeInfoProvider = ncltest.NewMockNodeInfoProvider()
	s.messageHandler = ncl.NewMockMessageHandler(s.ctrl)
	s.checkpointer = nclprotocol.NewMockCheckpointer(s.ctrl)
}

func (s *ConfigTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *ConfigTestSuite) getValidConfig() nclprotocolcompute.Config {
	return nclprotocolcompute.Config{
		NodeID: "test-node",
		ClientFactory: natsutil.ClientFactoryFunc(func(ctx context.Context) (*nats.Conn, error) {
			return nil, nil
		}),
		NodeInfoProvider:        s.nodeInfoProvider,
		MessageSerializer:       envelope.NewSerializer(),
		MessageRegistry:         nclprotocol.MustCreateMessageRegistry(),
		HeartbeatInterval:       time.Second,
		HeartbeatMissFactor:     5,
		NodeInfoUpdateInterval:  time.Second,
		RequestTimeout:          time.Second,
		ReconnectInterval:       time.Second,
		ReconnectBackoff:        backoff.NewExponential(time.Second, 2*time.Second),
		DataPlaneMessageHandler: s.messageHandler,
		DataPlaneMessageCreator: &ncltest.MockMessageCreator{},
		EventStore:              testutils.CreateComputeEventStore(s.T()),
		LogStreamServer:         &ncltest.MockLogStreamServer{},
		Checkpointer:            s.checkpointer,
		CheckpointInterval:      time.Second,
		Clock:                   clock.New(),
		DispatcherConfig:        dispatcher.DefaultConfig(),
	}
}

func (s *ConfigTestSuite) TestValidation() {
	testCases := []struct {
		name        string
		mutate      func(*nclprotocolcompute.Config)
		expectError string
	}{
		{
			name:        "valid config",
			mutate:      func(c *nclprotocolcompute.Config) {},
			expectError: "",
		},
		{
			name:        "missing node ID",
			mutate:      func(c *nclprotocolcompute.Config) { c.NodeID = "" },
			expectError: "nodeID cannot be blank",
		},
		{
			name:        "missing required dependencies",
			mutate:      func(c *nclprotocolcompute.Config) { c.ClientFactory = nil; c.NodeInfoProvider = nil },
			expectError: "cannot be nil",
		},
		{
			name: "invalid intervals",
			mutate: func(c *nclprotocolcompute.Config) {
				c.HeartbeatInterval = 0
				c.NodeInfoUpdateInterval = 0
			},
			expectError: "must be positive",
		},
		{
			name: "invalid dispatcher config",
			mutate: func(c *nclprotocolcompute.Config) {
				c.DispatcherConfig.StallTimeout = time.Second
				c.DispatcherConfig.StallCheckInterval = 2 * time.Second
			},
			expectError: "StallCheckInterval must be less than StallTimeout",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			cfg := s.getValidConfig()
			tc.mutate(&cfg)
			err := cfg.Validate()

			if tc.expectError == "" {
				s.NoError(err)
			} else {
				s.Error(err)
				s.Contains(err.Error(), tc.expectError)
			}
		})
	}
}

func (s *ConfigTestSuite) TestSetDefaults() {
	emptyConfig := nclprotocolcompute.Config{}
	emptyConfig.SetDefaults()
	defaults := nclprotocolcompute.DefaultConfig()

	s.Equal(defaults.HeartbeatMissFactor, emptyConfig.HeartbeatMissFactor)
	s.Equal(defaults.RequestTimeout, emptyConfig.RequestTimeout)
	s.NotNil(emptyConfig.MessageSerializer)
	s.NotNil(emptyConfig.ReconnectBackoff)
	s.NotEqual(dispatcher.Config{}, emptyConfig.DispatcherConfig)

	// Existing values should not be overwritten
	customConfig := nclprotocolcompute.Config{
		HeartbeatMissFactor: 10,
		RequestTimeout:      20 * time.Second,
	}
	customConfig.SetDefaults()
	s.Equal(10, customConfig.HeartbeatMissFactor)
	s.Equal(20*time.Second, customConfig.RequestTimeout)
}

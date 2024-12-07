package orchestrator

import (
	"errors"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	natsutil "github.com/bacalhau-project/bacalhau/pkg/nats"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/nodes"
	"github.com/bacalhau-project/bacalhau/pkg/transport"
	"github.com/bacalhau-project/bacalhau/pkg/transport/core"
)

type Config struct {
	NodeID        string
	ClientFactory natsutil.ClientFactory
	NodeManager   nodes.Manager

	MessageRegistry   *envelope.Registry
	MessageSerializer envelope.MessageSerializer

	// Control plane config
	HeartbeatTimeout      time.Duration
	NodeCleanupInterval   time.Duration
	RequestHandlerTimeout time.Duration

	// Data plane config
	DataPlaneMessageHandler        ncl.MessageHandler
	DataPlaneMessageCreatorFactory transport.MessageCreatorFactory
	EventStore                     watcher.EventStore
}

func (c *Config) Validate() error {
	return errors.Join(
		validate.NotBlank(c.NodeID, "node ID cannot be blank"),
		validate.NotNil(c.ClientFactory, "client factory cannot be nil"),
		validate.NotNil(c.NodeManager, "node manager cannot be nil"),
		validate.NotNil(c.MessageRegistry, "message registry cannot be nil"),
		validate.NotNil(c.MessageSerializer, "message serializer cannot be nil"),
		validate.IsGreaterThanZero(c.HeartbeatTimeout, "heartbeat timeout must be positive"),
		validate.IsGreaterThanZero(c.NodeCleanupInterval, "node cleanup interval must be positive"),
		validate.IsGreaterThanZero(c.RequestHandlerTimeout, "request handler timeout must be positive"),
		validate.NotNil(c.DataPlaneMessageHandler, "data plane message handler cannot be nil"),
		validate.NotNil(c.DataPlaneMessageCreatorFactory, "data plane message creator factory cannot be nil"),
		validate.NotNil(c.EventStore, "event store cannot be nil"),
	)
}

func DefaultConfig() Config {
	return Config{
		HeartbeatTimeout:      2 * time.Minute,
		NodeCleanupInterval:   30 * time.Second,
		RequestHandlerTimeout: 2 * time.Second,
		MessageSerializer:     envelope.NewSerializer(),
		MessageRegistry:       core.MustCreateMessageRegistry(),
	}
}

func (c *Config) setDefaults() {
	defaults := DefaultConfig()
	if c.HeartbeatTimeout == 0 {
		c.HeartbeatTimeout = defaults.HeartbeatTimeout
	}
	if c.NodeCleanupInterval == 0 {
		c.NodeCleanupInterval = defaults.NodeCleanupInterval
	}
	if c.RequestHandlerTimeout == 0 {
		c.RequestHandlerTimeout = defaults.RequestHandlerTimeout
	}
	if c.MessageSerializer == nil {
		c.MessageSerializer = defaults.MessageSerializer
	}
	if c.MessageRegistry == nil {
		c.MessageRegistry = defaults.MessageRegistry
	}
}

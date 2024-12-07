package compute

import (
	"errors"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/lib/backoff"
	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/nats"
	"github.com/bacalhau-project/bacalhau/pkg/transport"
	"github.com/bacalhau-project/bacalhau/pkg/transport/core"
	"github.com/bacalhau-project/bacalhau/pkg/transport/dispatcher"
)

type Config struct {
	NodeID           string
	ClientFactory    nats.ClientFactory
	NodeInfoProvider models.NodeInfoProvider

	MessageSerializer envelope.MessageSerializer
	MessageRegistry   *envelope.Registry

	// Control plane config
	ReconnectInterval      time.Duration
	HeartbeatInterval      time.Duration
	NodeInfoUpdateInterval time.Duration
	RequestTimeout         time.Duration
	Backoff                backoff.Backoff

	// Data plane config
	DataPlaneMessageHandler ncl.MessageHandler       // Handles incoming messages
	DataPlaneMessageCreator transport.MessageCreator // Creates messages for sending
	EventStore              watcher.EventStore
	DispatcherConfig        dispatcher.Config

	// Checkpoint config
	Checkpointer       core.Checkpointer
	CheckpointInterval time.Duration
	LogStreamServer    logstream.Server
}

// Validate checks if the config is valid
func (c *Config) Validate() error {
	return errors.Join(
		validate.NotBlank(c.NodeID, "nodeID cannot be blank"),
		validate.NotNil(c.ClientFactory, "client factory cannot be nil"),
		validate.NotNil(c.MessageSerializer, "message serializer cannot be nil"),
		validate.NotNil(c.MessageRegistry, "message registry cannot be nil"),
		validate.NotNil(c.NodeInfoProvider, "node info provider cannot be nil"),
		validate.NotNil(c.DataPlaneMessageHandler, "data plane message handler cannot be nil"),
		validate.NotNil(c.DataPlaneMessageCreator, "data plane message creator cannot be nil"),

		// validations for timing configs
		validate.IsGreaterThanZero(c.HeartbeatInterval, "heartbeat interval must be positive"),
		validate.IsGreaterThanZero(c.NodeInfoUpdateInterval, "node info update interval must be positive"),
		validate.IsGreaterThanZero(c.RequestTimeout, "request timeout must be positive"),
		validate.IsGreaterThanZero(c.ReconnectInterval, "reconnect interval must be positive"),
		validate.IsGreaterThanZero(c.CheckpointInterval, "checkpoint interval must be positive"),

		// validations for data plane components
		validate.NotNil(c.EventStore, "event store cannot be nil"),
		validate.NotNil(c.Backoff, "backoff cannot be nil"),
		validate.NotNil(c.Checkpointer, "checkpointer cannot be nil"),

		// Validate dispatcher config
		c.DispatcherConfig.Validate(),

		// Validate logical relationships between intervals
		validate.True(
			c.RequestTimeout < c.HeartbeatInterval,
			"request timeout must be less than heartbeat interval",
		),
		validate.True(
			c.HeartbeatInterval < c.NodeInfoUpdateInterval,
			"heartbeat interval should be less than node info update interval",
		),
	)
}

// DefaultConfig returns a new Config with default values
func DefaultConfig() Config {
	return Config{
		RequestTimeout:         5 * time.Second,
		ReconnectInterval:      5 * time.Second,
		HeartbeatInterval:      30 * time.Second,
		NodeInfoUpdateInterval: 1 * time.Minute,
		CheckpointInterval:     30 * time.Second,
		Backoff:                backoff.NewExponential(5*time.Second, 2*time.Minute),
		MessageSerializer:      envelope.NewSerializer(),
		MessageRegistry:        core.MustCreateMessageRegistry(),
		DispatcherConfig:       dispatcher.DefaultConfig(),
	}
}

func (c *Config) setDefaults() {
	defaults := DefaultConfig()
	if c.RequestTimeout == 0 {
		c.RequestTimeout = defaults.RequestTimeout
	}
	if c.ReconnectInterval == 0 {
		c.ReconnectInterval = defaults.ReconnectInterval
	}
	if c.HeartbeatInterval == 0 {
		c.HeartbeatInterval = defaults.HeartbeatInterval
	}
	if c.NodeInfoUpdateInterval == 0 {
		c.NodeInfoUpdateInterval = defaults.NodeInfoUpdateInterval
	}
	if c.CheckpointInterval == 0 {
		c.CheckpointInterval = defaults.CheckpointInterval
	}
	if c.MessageSerializer == nil {
		c.MessageSerializer = defaults.MessageSerializer
	}
	if c.MessageRegistry == nil {
		c.MessageRegistry = defaults.MessageRegistry
	}
	if c.Backoff == nil {
		c.Backoff = defaults.Backoff
	}
	if c.DispatcherConfig == (dispatcher.Config{}) {
		c.DispatcherConfig = defaults.DispatcherConfig
	}
}

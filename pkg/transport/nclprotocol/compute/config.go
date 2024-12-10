package compute

import (
	"errors"
	"time"

	"github.com/benbjohnson/clock"

	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/lib/backoff"
	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/nats"
	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol"
	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol/dispatcher"
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
	HeartbeatMissFactor    int
	NodeInfoUpdateInterval time.Duration
	RequestTimeout         time.Duration
	ReconnectBackoff       backoff.Backoff

	// Data plane config
	DataPlaneMessageHandler ncl.MessageHandler         // Handles incoming messages
	DataPlaneMessageCreator nclprotocol.MessageCreator // Creates messages for sending
	EventStore              watcher.EventStore
	DispatcherConfig        dispatcher.Config
	LogStreamServer         logstream.Server

	// Checkpoint config
	Checkpointer       nclprotocol.Checkpointer
	CheckpointInterval time.Duration

	Clock clock.Clock
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
		validate.IsGreaterThanZero(c.HeartbeatMissFactor, "heartbeat miss factor must be positive"),
		validate.IsGreaterThanZero(c.NodeInfoUpdateInterval, "node info update interval must be positive"),
		validate.IsGreaterThanZero(c.RequestTimeout, "request timeout must be positive"),
		validate.IsGreaterThanZero(c.ReconnectInterval, "reconnect interval must be positive"),
		validate.IsGreaterThanZero(c.CheckpointInterval, "checkpoint interval must be positive"),

		// validations for data plane components
		validate.NotNil(c.EventStore, "event store cannot be nil"),
		validate.NotNil(c.ReconnectBackoff, "backoff cannot be nil"),
		validate.NotNil(c.Checkpointer, "checkpointer cannot be nil"),

		// Validate dispatcher config
		c.DispatcherConfig.Validate(),
	)
}

// DefaultConfig returns a new Config with default values
//
//nolint:mnd
func DefaultConfig() Config {
	// defaults for heartbeatInterval and nodeInfoUpdateInterval are provided by BacalhauConfig,
	// and equal to 15 seconds and 1 minute respectively
	return Config{
		HeartbeatMissFactor: 5, // allow up to 5 missed heartbeats before marking a node as disconnected
		RequestTimeout:      10 * time.Second,
		ReconnectInterval:   10 * time.Second,
		CheckpointInterval:  30 * time.Second,
		ReconnectBackoff:    backoff.NewExponential(10*time.Second, 2*time.Minute),
		MessageSerializer:   envelope.NewSerializer(),
		MessageRegistry:     nclprotocol.MustCreateMessageRegistry(),
		DispatcherConfig:    dispatcher.DefaultConfig(),
		Clock:               clock.New(),
	}
}

func (c *Config) setDefaults() {
	defaults := DefaultConfig()
	if c.HeartbeatMissFactor == 0 {
		c.HeartbeatMissFactor = defaults.HeartbeatMissFactor
	}
	if c.RequestTimeout == 0 {
		c.RequestTimeout = defaults.RequestTimeout
	}
	if c.ReconnectInterval == 0 {
		c.ReconnectInterval = defaults.ReconnectInterval
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
	if c.ReconnectBackoff == nil {
		c.ReconnectBackoff = defaults.ReconnectBackoff
	}
	if c.DispatcherConfig == (dispatcher.Config{}) {
		c.DispatcherConfig = defaults.DispatcherConfig
	}
	if c.Clock == nil {
		c.Clock = defaults.Clock
	}
}

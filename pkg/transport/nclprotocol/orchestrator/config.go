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
	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol"
	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol/dispatcher"
)

// Config defines the configuration for the orchestrator's transport layer.
// It contains settings for both the control plane (node management, heartbeats)
// and data plane (message handling, event dispatching) components.
type Config struct {
	// NodeID uniquely identifies this orchestrator instance
	NodeID string

	// ClientFactory creates NATS clients for transport connections
	ClientFactory natsutil.ClientFactory

	// NodeManager handles compute node lifecycle and state management
	NodeManager nodes.Manager

	// Message serialization and type registration
	MessageRegistry   *envelope.Registry         // Registry of message types for serialization
	MessageSerializer envelope.MessageSerializer // Handles message envelope serialization

	// Control plane timeouts and intervals
	HeartbeatTimeout      time.Duration // Maximum time to wait for node heartbeat before considering it disconnected
	NodeCleanupInterval   time.Duration // How often to check for and cleanup disconnected nodes
	RequestHandlerTimeout time.Duration // Timeout for handling individual control plane requests

	// Data plane configuration
	DataPlaneMessageHandler        ncl.MessageHandler                // Handles incoming messages from compute nodes
	DataPlaneMessageCreatorFactory nclprotocol.MessageCreatorFactory // Creates message creators for outgoing messages
	EventStore                     watcher.EventStore                // Store for watching and dispatching events
	DispatcherConfig               dispatcher.Config                 // Configuration for the event dispatcher
}

// Validate checks if the configuration is valid by verifying:
// - Required fields are set
// - Timeouts and intervals are positive
// - Component configurations are valid
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

		// Validate nested dispatcher config
		c.DispatcherConfig.Validate(),
	)
}

func DefaultConfig() Config {
	return Config{
		// Default timeouts and intervals
		HeartbeatTimeout:      2 * time.Minute,  // Time before node considered disconnected
		NodeCleanupInterval:   30 * time.Second, // Check for disconnected nodes every 30s
		RequestHandlerTimeout: 2 * time.Second,  // Individual request timeout

		// Default message handling
		MessageSerializer: envelope.NewSerializer(),
		MessageRegistry:   nclprotocol.MustCreateMessageRegistry(),

		// Default dispatcher configuration
		DispatcherConfig: dispatcher.DefaultConfig(),
	}
}

// setDefaults applies default values to any unset fields in the config.
// It does not override values that are already set.
func (c *Config) setDefaults() {
	defaults := DefaultConfig()

	// Apply default timeouts if not set
	if c.HeartbeatTimeout == 0 {
		c.HeartbeatTimeout = defaults.HeartbeatTimeout
	}
	if c.NodeCleanupInterval == 0 {
		c.NodeCleanupInterval = defaults.NodeCleanupInterval
	}
	if c.RequestHandlerTimeout == 0 {
		c.RequestHandlerTimeout = defaults.RequestHandlerTimeout
	}

	// Apply default message handling if not set
	if c.MessageSerializer == nil {
		c.MessageSerializer = defaults.MessageSerializer
	}
	if c.MessageRegistry == nil {
		c.MessageRegistry = defaults.MessageRegistry
	}

	// Apply default dispatcher config if not set
	if c.DispatcherConfig == (dispatcher.Config{}) {
		c.DispatcherConfig = defaults.DispatcherConfig
	}
}

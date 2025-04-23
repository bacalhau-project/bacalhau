package analytics

import (
	"github.com/posthog/posthog-go"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
)

// TelemetryEndpoint is the endpoint to send telemetry data to.
// It's set at build time via ldflags for official builds.
var TelemetryEndpoint string = "" // Empty default means no telemetry

var (
	// posthogClient is the PostHog client for sending telemetry data.
	// This is a singleton and should be initialized once at startup.
	posthogClient posthog.Client

	// distinctID is the unique identifier for the node in telemetry events.
	// This is determined during Setup based on ResourceAttributes
	// and follows the priority order defined in DetermineDistinctID.
	distinctID = "unknown"
)

// Setup initializes the analytics provider with the provided configuration options.
// It creates a PostHog client configured with the resource attributes determined
// by the provided options.
//
// If TelemetryEndpoint is not set, Setup will not create a client and will return nil.
// This allows telemetry to be easily disabled.
//
// Returns an error if the client creation fails.
func Setup(opts ...Option) error {
	// Skip setup if telemetry endpoint is not set
	if TelemetryEndpoint == "" {
		log.Trace().Msg("Telemetry endpoint not set, skipping client setup")
		return nil
	}

	// Initialize resource attributes
	attributes := &ResourceAttributes{}

	// Apply all options
	for _, opt := range opts {
		opt(attributes)
	}

	// Apply defaults and fallbacks
	attributes.ApplyDefaults()

	// Create PostHog client with resource attributes as default properties
	client, err := posthog.NewWithConfig("", posthog.Config{
		Endpoint:               TelemetryEndpoint,
		DefaultEventProperties: posthog.Properties(attributes.Properties()),
	})
	if err != nil {
		return bacerrors.Newf("failed to create telemetry client: %s", err.Error()).
			WithComponent("analytics")
	}

	// Set the distinctID for all events based on resource attributes
	distinctID = attributes.DetermineDistinctID()

	// Store the client for later use
	posthogClient = client

	log.Debug().
		Str("distinctID", distinctID).
		Str("endpoint", TelemetryEndpoint).
		Msg("Telemetry client initialized")

	return nil
}

// Shutdown gracefully closes the analytics provider and releases resources.
// This should be called when the application is shutting down.
func Shutdown() {
	if posthogClient != nil {
		if err := posthogClient.Close(); err != nil {
			log.Trace().Err(err).Msg("Failed to shutdown PostHog client")
		}
	}
}

// Emit sends an analytics event to the telemetry backend.
// If the telemetry client is not initialized, this is a no-op.
//
// The event type and properties are determined by the provided Event.
// The distinctID is determined during Setup and used for all events.
func Emit(event Event) {
	// Skip if client is not initialized
	if posthogClient == nil {
		return
	}

	// Enqueue the event for sending
	if err := posthogClient.Enqueue(posthog.Capture{
		DistinctId: distinctID,
		Event:      event.Type(),
		Properties: posthog.Properties(event.Properties()),
	}); err != nil {
		log.Trace().
			Err(err).
			Str("type", event.Type()).
			Msg("Failed to emit telemetry event")
	}
}

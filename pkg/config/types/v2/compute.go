package v2

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/types/v2/executor"
	"github.com/bacalhau-project/bacalhau/pkg/config/types/v2/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/config/types/v2/storage"
	"github.com/bacalhau-project/bacalhau/pkg/config/types/v2/types"
)

// Compute represents the configuration for the compute service on the Bacalhau node.
// It includes settings for enabling the service, connecting to orchestrators, TLS, heartbeat, store, capacity, and more.
type Compute struct {
	// Enabled when set to true will enable the compute service on the Bacalhau node.
	Enabled bool
	// Orchestrators specifies a list of orchestrators the compute node will connect to.
	Orchestrators []string
	// TLS specifies the TLS configuration used to connect to orchestrators.
	TLS types.TLS
	// Labels specifies a map of key value pairs the compute node will advertise to orchestrators.
	Labels map[string]string
	// Heartbeat specifies the compute node's heartbeat configuration.
	Heartbeat Heartbeat
	// Store specifies the compute node's store configuration.
	Store ComputeStore
	// Capacity specifies the compute node's capacity configuration.
	Capacity Capacity
	// Publishers specifies the configuration of publishers the compute node provides.
	Publishers publisher.Providers
	// InputSources specifies the configuration of input sources the compute node provides.
	InputSources storage.Providers
	// Executors specifies the configuration of executors the compute node provides.
	Executors executor.Providers
	// Policy specifies the configuration of the compute node's job selection policy.
	Policy SelectionPolicy
}

// Heartbeat represents the configuration settings for the compute node's heartbeat messages.
type Heartbeat struct {
	// MessageInterval specifies the duration at which the compute node sends heartbeat messages to the orchestrators.
	MessageInterval types.Duration
	// ResourceInterval specifies the duration at which the compute node sends resource messages to the orchestrators.
	ResourceInterval types.Duration
	// InfoInterval specifies the duration at which the compute node sends info messages to the orchestrators.
	InfoInterval types.Duration
}

// ComputeStore represents the configuration settings for the compute node's storage backend.
type ComputeStore struct {
	// Type specifies the backend type of the ComputeStore. One of: boltdb.
	Type string
	// ExecutionGC specifies the garbage collection policy for executions in the ComputeStore.
	ExecutionGC types.TimeGC
}

// Capacity represents the capacity configuration settings for the compute node.
type Capacity struct {
	// Total when specified overrides the auto-detected capacity of the compute node.
	// When provided, the Allocated capacity will be ignored.
	Total types.ResourceConfig
	// Allocated specifies the percentage of the total capacity that can be allocated to jobs on the compute node.
	Allocated types.ResourceScalerConfig
}

// SelectionPolicy represents the job selection policy configuration for the compute node.
type SelectionPolicy struct {
	// Networked when set to true allows the compute node to accept jobs requiring network access.
	Networked bool
	// Local when set to true instructs the compute node to only accept jobs whose inputs it has locally.
	Local bool
}

package v2

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/types/v2/types"
)

// Orchestrator represents the configuration for the orchestration service on the Bacalhau node.
// It includes settings for enabling the service, network endpoints, TLS configuration, and various subsystems.
type Orchestrator struct {
	// Enabled specifies whether the orchestration service is enabled on the Bacalhau node.
	Enabled bool

	// Listen specifies the endpoint the orchestration service will listen on for connections from compute nodes.
	Listen string
	// Advertise specifies the endpoint the orchestration service will advertise to the network for connections from compute nodes.
	Advertise string
	// TLS specifies the TLS configuration of the orchestration service.
	TLS types.TLS

	// Cluster specifies the cluster configuration of the orchestration service.
	Cluster Cluster
	// NodeManager specifies the node manager configuration of the orchestration service.
	NodeManager NodeManager
	// Store specifies the store configuration of the orchestration service.
	Store OrchestratorStore
	// Scheduler specifies the scheduler configuration of the orchestration service.
	Scheduler Scheduler
	// Broker specifies the evaluation broker configuration of the orchestration service.
	Broker EvaluationBroker
}

// Cluster represents the configuration settings for the orchestration service NATs cluster.
type Cluster struct {
	// Listen specifies the address the orchestration service will listen on for connections from other orchestration services.
	Listen string
	// Advertise specifies the endpoint the orchestration service will advertise to the network for connections from other orchestration services.
	Advertise string
	// Peers specifies the list of peer orchestration services.
	Peers []string
	// TLS specifies the TLS configuration for connections from orchestration services.
	TLS types.TLS
}

// NodeManager represents the configuration settings for the node manager within the orchestration service.
type NodeManager struct {
	// DisconnectTimeout specifies the duration after which nodes will be considered disconnected if no heartbeat message is received.
	DisconnectTimeout types.Duration
	// AutoApprove specifies whether to automatically approve a node's membership when a connection is established.
	// When set to false, sets a node's approval status to 'pending' when a connection is established.
	AutoApprove bool
	// GC specifies the garbage collection configuration of the node manager.
	GC types.TimeGC
}

// OrchestratorStore represents the configuration settings for the storage backend of the orchestration service.
type OrchestratorStore struct {
	// Type specifies the backend type of the orchestrator store. One of: boltdb.
	Type string
	// JobGC specifies the garbage collection policy for jobs in the orchestrator store.
	JobGC types.TimeGC
	// EvaluationGC specifies the garbage collection policy for evaluations in the orchestrator store.
	EvaluationGC types.TimeGC
}

// Scheduler represents the configuration settings for the scheduler within the orchestration service.
type Scheduler struct {
	// Workers specifies the number of workers the scheduler will run.
	Workers int
	// HousekeepingInterval specifies the interval at which housekeeping tasks are performed.
	HousekeepingInterval types.Duration
	// HousekeepingTimeout specifies the timeout duration for housekeeping tasks.
	HousekeepingTimeout types.Duration
}

// EvaluationBroker represents the configuration settings for the evaluation broker within the orchestration service.
type EvaluationBroker struct {
	// VisibilityTimeout specifies the duration after which an unprocessed evaluation is re-queued.
	VisibilityTimeout types.Duration
	// MaxRetries specifies the maximum number of retries for processing an evaluation.
	MaxRetries int
}

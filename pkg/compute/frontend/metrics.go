package frontend

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics for monitoring compute nodes:
var (
	jobsReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "jobs_received",
			Help: "Number of jobs received by the compute node.",
		},
		[]string{"node_id", "client_id"},
	)

	jobsAccepted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "jobs_accepted",
			Help: "Number of jobs bid on and accepted by the compute node.",
		},
		[]string{"node_id", "shard_index", "client_id"},
	)
)

package backend

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics for monitoring compute nodes:
var (
	jobsCompleted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "jobs_completed",
			Help: "Number of jobs completed by the compute node.",
		},
		[]string{"node_id", "shard_index", "client_id"},
	)

	jobsFailed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "jobs_failed",
			Help: "Number of jobs failed by the compute node.",
		},
		[]string{"node_id", "shard_index", "client_id"},
	)
)

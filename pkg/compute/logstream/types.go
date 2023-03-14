package logstream

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/libp2p/go-libp2p/core/host"
)

type LogStreamServer struct {
	Address        string
	host           host.Host
	ctx            context.Context
	executionStore store.ExecutionStore
	executors      executor.ExecutorProvider
}

type LogStreamRequest struct {
	JobID       string
	ExecutionID string
	WithHistory bool
}

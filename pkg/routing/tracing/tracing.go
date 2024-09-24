package tracing

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
)

type NodeStore struct {
	delegate routing.NodeInfoStore
}

func NewNodeStore(delegate routing.NodeInfoStore) *NodeStore {
	return &NodeStore{
		delegate: delegate,
	}
}

func (r *NodeStore) Add(ctx context.Context, state models.NodeState) error {
	ctx, span := telemetry.NewSpan(ctx, telemetry.GetTracer(), "pkg/routing.NodeInfoStore.Add") //nolint:govet
	defer span.End()

	stopwatch := telemetry.Timer(ctx, addNodeDurationMilliseconds)
	defer func() {
		dur := stopwatch()
		log.Ctx(ctx).Trace().
			Dur("duration", dur).
			Str("node", state.Info.ID()).
			Msg("node added")
	}()

	return r.delegate.Add(ctx, state)
}

func (r *NodeStore) Get(ctx context.Context, nodeID string) (models.NodeState, error) {
	ctx, span := telemetry.NewSpan(ctx, telemetry.GetTracer(), "pkg/routing.NodeInfoStore.Get") //nolint:govet
	defer span.End()

	stopwatch := telemetry.Timer(ctx, getNodeDurationMilliseconds)
	defer func() {
		dur := stopwatch()
		log.Ctx(ctx).Trace().
			Dur("duration", dur).
			Str("node", nodeID).
			Msg("node retrieved")
	}()

	return r.delegate.Get(ctx, nodeID)
}

func (r *NodeStore) GetByPrefix(ctx context.Context, prefix string) (models.NodeState, error) {
	ctx, span := telemetry.NewSpan(ctx, telemetry.GetTracer(), "pkg/routing.NodeInfoStore.GetByPrefix") //nolint:govet
	defer span.End()

	stopwatch := telemetry.Timer(ctx, getPrefixNodeDurationMilliseconds)
	defer func() {
		dur := stopwatch()
		log.Ctx(ctx).Trace().
			Dur("duration", dur).
			Str("prefix", prefix).
			Msg("node retrieved by previus")
	}()

	return r.delegate.GetByPrefix(ctx, prefix)
}

func (r *NodeStore) List(ctx context.Context, filters ...routing.NodeStateFilter) ([]models.NodeState, error) {
	ctx, span := telemetry.NewSpan(ctx, telemetry.GetTracer(), "pkg/routing.NodeInfoStore.List") //nolint:govet
	defer span.End()

	stopwatch := telemetry.Timer(ctx, listNodesDurationMilliseconds)
	defer func() {
		dur := stopwatch()
		log.Ctx(ctx).Trace().
			Dur("duration", dur).
			Msg("node listed")
	}()

	return r.delegate.List(ctx, filters...)
}

func (r *NodeStore) Delete(ctx context.Context, nodeID string) error {
	ctx, span := telemetry.NewSpan(ctx, telemetry.GetTracer(), "pkg/routing.NodeInfoStore.Delete") //nolint:govet
	defer span.End()

	stopwatch := telemetry.Timer(ctx, deleteNodeDurationMilliseconds)
	defer func() {
		dur := stopwatch()
		log.Ctx(ctx).Trace().
			Dur("duration", dur).
			Str("node", nodeID).
			Msg("node deleted")
	}()

	return r.delegate.Delete(ctx, nodeID)
}

// compile time check that we implement the interface
var _ routing.NodeInfoStore = (*NodeStore)(nil)

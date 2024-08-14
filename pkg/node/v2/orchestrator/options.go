package orchestrator

import (
	"github.com/bacalhau-project/bacalhau/pkg/eventhandler"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/node/manager"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
)

type Option func(*Node) error

func WithScheduler(s *SchedulerService) Option {
	return func(n *Node) error {
		n.Scheduler = s
		return nil
	}
}

func WithEndpoint(e *requester.BaseEndpoint) Option {
	return func(n *Node) error {
		n.Endpoint = e
		return nil
	}
}

func WithTracerContextProvider(t *eventhandler.TracerContextProvider) Option {
	return func(n *Node) error {
		n.TracerContextProvider = t
		return nil
	}
}

func WithDebugInfoProvider(p ...models.DebugInfoProvider) Option {
	return func(n *Node) error {
		n.DebugInfoProvider = p
		return nil
	}
}

func WithNodeInfoStore(s routing.NodeInfoStore) Option {
	return func(n *Node) error {
		n.NodeInfoStore = s
		return nil
	}
}

func WithNodeManer(m *manager.NodeManager) Option {
	return func(n *Node) error {
		n.NodeManager = m
		return nil
	}
}

func WithJobStore(s jobstore.Store) Option {
	return func(n *Node) error {
		n.Store = s
		return nil
	}
}

func WithEventTracer(t *eventhandler.Tracer) Option {
	return func(n *Node) error {
		n.EventTracer = t
		return nil
	}
}

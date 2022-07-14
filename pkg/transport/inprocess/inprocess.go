package inprocess

import (
	"context"
	"fmt"
	"sync"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/transport"
	"github.com/google/uuid"
)

// Transport is a transport layer that operates entirely in-memory, for
// testing purposes. Should not be used in production.
type InProcessTransport struct {
	id                 string
	subscribeFunctions []transport.SubscribeFn
	ctx                sync.Mutex
}

/*

  lifecycle

*/

func NewInprocessTransport() (*InProcessTransport, error) {
	hostID, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("inprocess: error creating host.id: %w", err)
	}
	res := &InProcessTransport{
		id:                 hostID.String(),
		subscribeFunctions: []transport.SubscribeFn{},
	}
	return res, nil
}

func (t *InProcessTransport) Start(ctx context.Context) error {
	if len(t.subscribeFunctions) == 0 {
		panic("Programming error: no subscribe func, please call Subscribe immediately after constructing interface")
	}
	return nil
}

func (t *InProcessTransport) Shutdown(ctx context.Context) error {
	return nil
}

func (t *InProcessTransport) HostID(ctx context.Context) (string, error) {
	return t.id, nil
}

/*

  pub / sub

*/

func (t *InProcessTransport) Publish(ctx context.Context, ev executor.JobEvent) error {
	t.ctx.Lock()
	defer t.ctx.Unlock()
	for _, fn := range t.subscribeFunctions {
		go fn(ctx, ev)
	}
	return nil
}

func (t *InProcessTransport) Subscribe(fn transport.SubscribeFn) {
	t.ctx.Lock()
	defer t.ctx.Unlock()
	t.subscribeFunctions = append(t.subscribeFunctions, fn)
}

// Static check to ensure that InProcessTransport implements Transport:
var _ transport.Transport = (*InProcessTransport)(nil)

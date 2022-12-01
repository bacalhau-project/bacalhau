package inprocess

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/rs/zerolog/log"

	sync "github.com/lukemarsden/golang-mutex-tracer"
	"github.com/multiformats/go-multiaddr"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/transport"
	"github.com/google/uuid"
)

// Transport is a transport layer that operates entirely in-memory, for
// testing purposes. Should not be used in production.
type InProcessTransport struct {
	id                 string
	subscribeFunctions []transport.SubscribeFn
	seenEvents         []model.JobEvent
	mutex              sync.Mutex
	// if a publishHandler is assigned - it will take over
	// from the standard publish - this is used by the InProcessTransportCluster
	// to hijack publish calls and distribute them across all the other nodes
	// (a bit like the libp2p transport where every event is written to the
	// event handlers of every node)
	publishHandler func(ctx context.Context, ev model.JobEvent) error
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
	res.mutex.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "InprocessTransport.mutex",
	})
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

func (t *InProcessTransport) HostID() string {
	return t.id
}

func (t *InProcessTransport) HostAddrs() ([]multiaddr.Multiaddr, error) {
	return []multiaddr.Multiaddr{}, nil
}

func (t *InProcessTransport) GetEvents() []model.JobEvent {
	return t.seenEvents
}

/*

  pub / sub

*/

func (t *InProcessTransport) Publish(ctx context.Context, ev model.JobEvent) error {
	if t.publishHandler != nil {
		// we have been given an external function to call with our event
		return t.publishHandler(ctx, ev)
	} else {
		// we will handle our event internally
		return t.applyEvent(ctx, ev)
	}
}

func (t *InProcessTransport) Subscribe(ctx context.Context, fn transport.SubscribeFn) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.subscribeFunctions = append(t.subscribeFunctions, fn)
}

/*
encrypt / decrypt
*/

func (*InProcessTransport) Encrypt(ctx context.Context, data, encryptionKeyBytes []byte) ([]byte, error) {
	return data, nil
}

func (*InProcessTransport) Decrypt(ctx context.Context, data []byte) ([]byte, error) {
	return data, nil
}

func (t *InProcessTransport) applyEvent(ctx context.Context, ev model.JobEvent) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	ctx = logger.ContextWithNodeIDLogger(context.Background(), t.HostID())
	t.seenEvents = append(t.seenEvents, ev)
	for _, fn := range t.subscribeFunctions {
		fnToCall := fn
		go func() {
			err := fnToCall(ctx, ev)
			if err != nil {
				log.Error().Msgf("error in handle event: %s\n%+v", err, ev)
			}
		}()
	}
	return nil
}

// Static check to ensure that InProcessTransport implements Transport:
var _ transport.Transport = (*InProcessTransport)(nil)

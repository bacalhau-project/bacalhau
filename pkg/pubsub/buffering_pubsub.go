package pubsub

import (
	"context"
	"fmt"
	"time"

	sync "github.com/lukemarsden/golang-mutex-tracer"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

const shutdownFlushTimeout = 5 * time.Second

type BufferingEnvelope struct {
	Offsets  []int64 `json:"offsets"`
	Payloads []byte  `json:"payloads"`
}

// Size return the size of the buffered payloads in the envelope
func (e *BufferingEnvelope) Size() int64 {
	return int64(len(e.Payloads))
}

type BufferingPubSubParams struct {
	DelegatePubSub PubSub[BufferingEnvelope]
	MaxBufferSize  int64
	MaxBufferAge   time.Duration
}

// BufferingPubSub is a PubSub implementation that buffers messages in memory and flushes them to the delegate PubSub
// when the buffer is full or the buffer age is reached
type BufferingPubSub[T any] struct {
	SingletonPubSub[T]
	delegatePubSub       PubSub[BufferingEnvelope]
	flushMutex           sync.RWMutex
	maxBufferSize        int64
	maxBufferAge         time.Duration
	currentBuffer        BufferingEnvelope
	oldestMessageTime    time.Time
	antiStarvationTicker *time.Ticker
}

func NewBufferingPubSub[T any](cm *system.CleanupManager, params BufferingPubSubParams) *BufferingPubSub[T] {
	newPubSub := &BufferingPubSub[T]{
		delegatePubSub: params.DelegatePubSub,
		maxBufferSize:  params.MaxBufferSize,
		maxBufferAge:   params.MaxBufferAge,
	}

	newPubSub.resetBuffer()
	newPubSub.delegatePubSub.Subscribe(context.Background(), newPubSub)
	newPubSub.flushMutex.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "BufferingPubSub.flushMutex",
	})

	// Start a background task to flush the buffer if it gets too old, and avoid situations where no more messages were published for a
	// a long time and the buffer is never flushed
	antiStarvationCtx, cancel := context.WithCancel(context.Background())
	go newPubSub.antiStarvationTask(antiStarvationCtx)
	cm.RegisterCallback(func() error {
		cancel()
		return nil
	})
	return newPubSub
}

func (p *BufferingPubSub[T]) Publish(ctx context.Context, message T) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/pubsub/Buffering.Publish")
	defer span.End()

	payload, err := model.JSONMarshalWithMax(message)
	if err != nil {
		return err
	}

	p.flushMutex.Lock()
	defer p.flushMutex.Unlock()
	if p.currentBuffer.Size() == 0 {
		p.oldestMessageTime = time.Now()
	}
	p.currentBuffer.Offsets = append(p.currentBuffer.Offsets, p.currentBuffer.Size())
	p.currentBuffer.Payloads = append(p.currentBuffer.Payloads, payload...)

	if p.currentBuffer.Size() >= p.maxBufferSize || time.Since(p.oldestMessageTime) > p.maxBufferAge {
		err = p.flushBuffer(ctx)
		p.resetBuffer()
	}

	return err
}

func (p *BufferingPubSub[T]) Handle(ctx context.Context, envelope BufferingEnvelope) error {
	for i, offset := range envelope.Offsets {
		var payload []byte
		if i == len(envelope.Offsets)-1 {
			payload = envelope.Payloads[offset:]
		} else {
			payload = envelope.Payloads[offset:envelope.Offsets[i+1]]
		}

		var message T
		err := model.JSONUnmarshalWithMax(payload, &message)
		if err != nil {
			return fmt.Errorf("failed to unmarshal message: %w", err)
		}
		if p.Subscriber != nil {
			err := p.Subscriber.Handle(ctx, message)
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("failed to handle message. Continuing")
			}
		} else {
			log.Ctx(ctx).Warn().Msg("no subscriber registered for Buffering pubsub. ignoring message")
			return nil
		}
	}
	return nil
}

// flush the buffer to the delegate pubsub
func (p *BufferingPubSub[T]) flushBuffer(ctx context.Context) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/pubsub/Buffering.Publish")
	defer span.End()

	log.Ctx(ctx).Trace().Msgf("flushing pubsub buffer after %s with %d messages, %d bytes",
		time.Since(p.oldestMessageTime), len(p.currentBuffer.Offsets), p.currentBuffer.Size())
	return p.delegatePubSub.Publish(ctx, p.currentBuffer)
}

// reset the buffer
func (p *BufferingPubSub[T]) resetBuffer() {
	p.currentBuffer = BufferingEnvelope{}
}

func (p *BufferingPubSub[T]) antiStarvationTask(ctx context.Context) {
	p.antiStarvationTicker = time.NewTicker(p.maxBufferAge)

	flush := func(fCtx context.Context) {
		p.flushMutex.Lock()
		defer p.flushMutex.Unlock()
		err := p.flushBuffer(fCtx)
		p.resetBuffer()
		if err != nil {
			// use original context here for logging purposes
			log.Ctx(ctx).Error().Err(err).Msg("failed to flush buffer")
		}
	}

	for {
		select {
		case <-p.antiStarvationTicker.C:
			if p.currentBuffer.Size() > 0 && time.Since(p.oldestMessageTime) > p.maxBufferAge {
				flush(ctx)
			}
		case <-ctx.Done():
			// flush the buffer before exiting
			// TODO: this still fails as libp2p ctx will also be closed by cleanup manager. We need a better way to sequence cleaning up
			//  of resources and stopping the background tasks.
			if p.currentBuffer.Size() > 0 {
				timeoutCtx, cancel := context.WithTimeout(context.Background(), shutdownFlushTimeout)
				flush(timeoutCtx)
				cancel()
			}
		}
	}
}

// compile-time interface assertions
var _ PubSub[string] = (*BufferingPubSub[string])(nil)

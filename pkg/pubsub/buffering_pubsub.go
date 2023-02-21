package pubsub

import (
	"context"
	"errors"
	"fmt"
	realsync "sync"
	"time"

	sync "github.com/bacalhau-project/golang-mutex-tracer"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

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
	delegatePubSub PubSub[BufferingEnvelope]
	maxBufferSize  int64
	maxBufferAge   time.Duration

	subscriber     Subscriber[T]
	subscriberOnce realsync.Once
	closeOnce      realsync.Once

	currentBuffer     BufferingEnvelope
	oldestMessageTime time.Time
	flushMutex        sync.RWMutex

	antiStarvationTicker *time.Ticker
	antiStarvationStop   chan struct{}
}

func NewBufferingPubSub[T any](params BufferingPubSubParams) *BufferingPubSub[T] {
	newPubSub := &BufferingPubSub[T]{
		delegatePubSub:     params.DelegatePubSub,
		maxBufferSize:      params.MaxBufferSize,
		maxBufferAge:       params.MaxBufferAge,
		antiStarvationStop: make(chan struct{}),
	}

	newPubSub.flushMutex.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "BufferingPubSub.flushMutex",
	})

	return newPubSub
}

func (p *BufferingPubSub[T]) Publish(ctx context.Context, message T) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/pubsub.BufferingPubSub.Publish")
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
		go p.flushBuffer(ctx, p.currentBuffer, p.oldestMessageTime)
		p.currentBuffer = BufferingEnvelope{} // reset the buffer
	}

	return err
}

func (p *BufferingPubSub[T]) Subscribe(ctx context.Context, subscriber Subscriber[T]) (err error) {
	var firstSubscriber bool
	p.subscriberOnce.Do(func() {
		// register the subscriber
		p.subscriber = subscriber

		// subscribe to the delegate pubsub
		err = p.delegatePubSub.Subscribe(ctx, p)

		// start the anti-starvation background task
		go p.antiStarvationTask()
		firstSubscriber = true
	})
	if err != nil {
		return err
	}
	if !firstSubscriber {
		err = errors.New("only a single subscriber is allowed. Use ChainedSubscriber to chain multiple subscribers")
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
		if p.subscriber != nil {
			err := p.subscriber.Handle(ctx, message)
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

func (p *BufferingPubSub[T]) Close(ctx context.Context) (err error) {
	p.closeOnce.Do(func() {
		if p.antiStarvationTicker != nil {
			// stop the anti-starvation background task
			p.antiStarvationStop <- struct{}{}
		}

		// flush the buffer before closing
		if p.currentBuffer.Size() > 0 {
			p.flushMutex.Lock()
			defer p.flushMutex.Unlock()
			p.flushBuffer(ctx, p.currentBuffer, p.oldestMessageTime)
			p.currentBuffer = BufferingEnvelope{} // reset the buffer
		}
	})
	if err != nil {
		return err
	}
	log.Ctx(ctx).Info().Msg("done closing BufferingPubSub")
	return nil
}

// flush the buffer to the delegate pubsub
func (p *BufferingPubSub[T]) flushBuffer(ctx context.Context, envelope BufferingEnvelope, oldestMessageTime time.Time) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/pubsub.BufferingPubSub.flushBuffer")
	defer span.End()

	log.Ctx(ctx).Trace().Msgf("flushing pubsub buffer after %s with %d messages, %d bytes",
		time.Since(oldestMessageTime), len(envelope.Offsets), envelope.Size())
	err := p.delegatePubSub.Publish(ctx, envelope)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to publish buffered messages")
	}
}

func (p *BufferingPubSub[T]) antiStarvationTask() {
	ctx := context.Background()
	p.antiStarvationTicker = time.NewTicker(p.maxBufferAge)

	for {
		select {
		case <-p.antiStarvationTicker.C:
			if p.currentBuffer.Size() > 0 && time.Since(p.oldestMessageTime) > p.maxBufferAge {
				func() {
					ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/pubsub.BufferingPubSub.antiStarvationTask") //nolint:govet
					defer span.End()
					p.flushMutex.Lock()
					defer p.flushMutex.Unlock()
					go p.flushBuffer(ctx, p.currentBuffer, p.oldestMessageTime)
					p.currentBuffer = BufferingEnvelope{} // reset the buffer
				}()
			}
		case <-p.antiStarvationStop:
			// do nothing as Close() will flush the buffer
			p.antiStarvationTicker.Stop()
			return
		}
	}
}

// compile-time interface assertions
var _ PubSub[string] = (*BufferingPubSub[string])(nil)

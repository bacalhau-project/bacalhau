package ncl

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
)

const (
	queueSize = 64
)

type orderedPublisher struct {
	*publisher
	config OrderedPublisherConfig

	queue         chan *pendingMsg
	inflight      sync.Map // map[string]*pendingMsg // Use msgID as key instead of reply subject
	inflightCount atomic.Int32

	inbox        string
	subscription *nats.Subscription
	shutdown     chan struct{}
	reset        chan struct{}
	resetDone    chan struct{}
	wg           sync.WaitGroup
	mu           sync.Mutex
}

type pendingMsg struct {
	msg      *nats.Msg
	future   *pubFuture
	ctx      context.Context
	timeout  time.Time
	enqueued time.Time
}

func NewOrderedPublisher(nc *nats.Conn, config OrderedPublisherConfig) (OrderedPublisher, error) {
	config.setDefaults()

	parentPublisher, err := NewPublisher(nc, *config.toPublisherConfig())
	if err != nil {
		return nil, err
	}

	p := &orderedPublisher{
		publisher: parentPublisher.(*publisher),
		config:    config,
		queue:     make(chan *pendingMsg, queueSize),
		shutdown:  make(chan struct{}),
		reset:     make(chan struct{}),
		resetDone: make(chan struct{}),
	}

	// Validate
	if err := p.validate(); err != nil {
		return nil, fmt.Errorf("invalid ordered publisher config: %w", err)
	}

	// Always need publishLoop
	p.wg.Add(1)
	go p.publishLoop()

	// Start timeoutLoop and subscribe to inbox if ack mode is enabled
	if config.AckMode != NoAck {
		// Subscribe to responses
		p.inbox = nc.NewInbox()
		sub, err := nc.Subscribe(p.inbox+".*", p.handleResponse)
		if err != nil {
			return nil, fmt.Errorf("failed to create response subscription: %w", err)
		}
		p.subscription = sub

		// Start timeout loop
		p.wg.Add(1)
		go p.timeoutLoop()
	}

	// Register meter callbacks
	_, err = Meter.RegisterCallback(
		func(_ context.Context, o metric.Observer) error {
			opts := metric.WithAttributes(attribute.String(AttrInstance, p.config.Name))
			o.ObserveInt64(asyncPublishInflightGauge, int64(p.inflightCount.Load()), opts)
			o.ObserveInt64(asyncPublishQueueGauge, int64(len(p.queue)), opts)
			return nil
		}, asyncPublishInflightGauge, asyncPublishQueueGauge)
	if err != nil {
		return nil, fmt.Errorf("failed to register metric callback: %w", err)
	}

	return p, nil
}

func (p *orderedPublisher) validate() error {
	return errors.Join(
		p.publisher.validate(),
		p.config.Validate(),
	)
}

// Publish publishes synchronously by waiting for async result
func (p *orderedPublisher) Publish(ctx context.Context, request PublishRequest) error {
	future, err := p.PublishAsync(ctx, request)
	if err != nil {
		return err
	}

	select {
	case <-future.Done():
		return future.Err()
	case <-ctx.Done():
		return ctx.Err()
	}
}

// PublishAsync queues message for publishing and returns a future
func (p *orderedPublisher) PublishAsync(ctx context.Context, request PublishRequest) (_ PubFuture, err error) {
	metrics := telemetry.NewMetricRecorder(
		attribute.String(AttrInstance, p.config.Name),
		attribute.String(AttrOutcome, OutcomeSuccess),
	)
	defer func() {
		if err != nil {
			metrics.Error(err)
			metrics.WithAttributes(attribute.String(AttrOutcome, OutcomeFailure))
		}
		metrics.Count(ctx, publishCount)
		metrics.Count(ctx, asyncPublishEnqueueCount)
		metrics.Done(ctx, asyncPublishEnqueueDuration)
	}()

	if p.pendingCount() >= p.config.MaxPending {
		return nil, fmt.Errorf("publisher max pending messages reached")
	}
	if err = p.validateRequest(request); err != nil {
		return nil, err
	}

	msg, err := p.encodeMsg(request)
	if err != nil {
		return nil, err
	}
	metrics.Latency(ctx, asyncPublishEnqueuePartDuration, "encode")
	metrics.Histogram(ctx, publishBytes, float64(len(msg.Data)))

	p.mu.Lock()
	defer p.mu.Unlock()
	metrics.Latency(ctx, asyncPublishEnqueuePartDuration, "lock")

	future := newPubFuture(msg)
	now := time.Now()
	pMsg := &pendingMsg{
		msg:      msg,
		future:   future,
		ctx:      ctx,
		enqueued: now,
		timeout:  now.Add(p.config.AckWait),
	}

	defer metrics.Latency(ctx, asyncPublishEnqueuePartDuration, "enqueue")
	select {
	case p.queue <- pMsg:
		return future, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return nil, fmt.Errorf("publisher max inflight reached")
	}
}

func (p *orderedPublisher) Reset(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()

	log.Info().
		Str("publisher", p.config.Name).
		Msg("publisher reset - clearing pending messages")

	// Signal reset to publishLoop
	p.reset <- struct{}{}
	p.drain(errors.New("publisher reset"))
	p.resetDone <- struct{}{}
}

func (p *orderedPublisher) Close(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	select {
	case <-p.shutdown:
		// Already closed
		return nil
	default:
		close(p.shutdown)
	}

	if p.subscription != nil {
		if err := p.subscription.Unsubscribe(); err != nil {
			return fmt.Errorf("failed to unsubscribe: %w", err)
		}
	}

	// create a done channel to wait for the publisher to finish
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	// wait for the publisher to finish or the context to be done
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// pendingCount returns the number of pending messages
func (p *orderedPublisher) pendingCount() int {
	return len(p.queue) + int(p.inflightCount.Load())
}

func (p *orderedPublisher) publishLoop() {
	defer p.wg.Done()

	for {
		select {
		case pubMsg := <-p.queue:
			p.processMessage(pubMsg)
		case <-p.reset:
			// wait for reset before continue processing
			<-p.resetDone
			continue
		case <-p.shutdown:
			p.drain(errors.New("publisher shutdown"))
			return
		}
	}
}

func (p *orderedPublisher) timeoutLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.AckWait / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx := context.Background()
			metrics := telemetry.NewMetricRecorder(
				attribute.String(AttrInstance, p.config.Name),
			)
			timeoutCount := 0
			now := time.Now()
			p.inflight.Range(func(key, value interface{}) bool {
				pMsg := value.(*pendingMsg)
				if now.After(pMsg.timeout) {
					timeoutCount++
					pMsg.future.setError(errors.New("publish ack timeout"))
					p.inflight.Delete(key)
					p.inflightCount.Add(-1)
				}
				// TODO: Could implement retry logic here
				return true
			})
			if timeoutCount > 0 {
				metrics.CountN(ctx, asyncPublishTimeoutCount, int64(timeoutCount))
			}
			metrics.Done(ctx, asyncPublishTimeoutLoopDuration)
		case <-p.shutdown:
			return
		}
	}
}

// encodeMsg to override the publisher's create message and add reply subject
func (p *orderedPublisher) encodeMsg(request PublishRequest) (*nats.Msg, error) {
	msg, err := p.publisher.encodeMsg(request)
	if err != nil {
		return nil, err
	}

	// Generate unique reply subject, if we expect an ack
	if p.config.AckMode != NoAck {
		msg.Reply = p.inbox + "." + uuid.NewString()
	}
	return msg, nil
}

func (p *orderedPublisher) processMessage(pubMsg *pendingMsg) {
	ctx := pubMsg.ctx
	metrics := telemetry.NewMetricRecorder(
		attribute.String(AttrInstance, p.config.Name),
		attribute.String(AttrOutcome, OutcomeSuccess),
	)
	defer func() {
		if pubMsg.future.Err() != nil {
			metrics.Error(pubMsg.future.Err())
			metrics.WithAttributes(attribute.String(AttrOutcome, OutcomeFailure))
		}
		metrics.Count(ctx, asyncPublishProcessCount)
		metrics.Done(ctx, asyncPublishProcessDuration)
	}()

	// Check if the publish context is still valid
	if err := pubMsg.ctx.Err(); err != nil {
		pubMsg.future.setError(err)
		return
	}

	// check if the msg hasn't been cancelled
	if pubMsg.future.Err() != nil {
		metrics.WithAttributes(attribute.String(AttrOutcome, OutcomeCancelled))
		return
	}

	// Publish with reply subject
	if err := p.nc.PublishMsg(pubMsg.msg); err != nil {
		pubMsg.future.setError(err)
		return
	}
	metrics.Latency(ctx, asyncPublishProcessPartDuration, "publish")

	// If no ack mode, set result
	if p.config.AckMode == NoAck {
		pubMsg.future.setResult(&Result{})
		publishDuration.Record(ctx, time.Since(pubMsg.enqueued).Seconds(), metrics.AttributesOpt())
	} else {
		p.inflight.Store(pubMsg.msg.Reply, pubMsg)
		p.inflightCount.Add(1)
	}
}

func (p *orderedPublisher) handleResponse(msg *nats.Msg) {
	ctx := context.Background()
	metrics := telemetry.NewMetricRecorder(
		attribute.String(AttrInstance, p.config.Name),
		attribute.String(AttrOutcome, OutcomeSuccess),
	)
	var future *pubFuture
	defer func() {
		if future != nil && future.Err() != nil {
			metrics.Error(future.Err())
			metrics.WithAttributes(attribute.String(AttrOutcome, OutcomeFailure))
		}
		metrics.Count(ctx, asyncPublishCallbackCount)
		metrics.Done(ctx, asyncPublishCallbackDuration)
	}()

	// Look up pending future
	val, ok := p.inflight.Load(msg.Subject)
	if !ok {
		// Response for unknown request - ignore after logging
		log.Debug().Str("subject", msg.Subject).Msg("received response for unknown request")
		metrics.ErrorString("unknown_response")
		metrics.WithAttributes(attribute.String(AttrOutcome, OutcomeFailure))
		return
	}
	metrics.Latency(ctx, asyncPublishCallbackPartDuration, "lookup")

	defer func() {
		p.inflight.Delete(msg.Subject)
		p.inflightCount.Add(-1)
	}()

	pMsg := val.(*pendingMsg)
	future = pMsg.future

	res, err := ParseResult(msg)
	if err != nil {
		future.setError(err)
		return
	}
	metrics.Latency(ctx, asyncPublishCallbackPartDuration, "decode")

	if res.Error != "" {
		// TODO: requeue with backoff, but keep track of retries
		future.setError(res.Err())
	} else {
		future.setResult(res)
	}
	publishDuration.Record(ctx, time.Since(pMsg.enqueued).Seconds(), metrics.AttributesOpt())
}

func (p *orderedPublisher) drain(err error) {
	log.Debug().
		Int32("inflight_msgs", p.inflightCount.Load()).
		Int("queued_msgs", len(p.queue)).
		Msgf("draining publisher queue due to: %v", err)

	for {
		select {
		case pubMsg := <-p.queue:
			pubMsg.future.setError(err)
		default:
			goto drainInflight
		}
	}

drainInflight:
	// then, mark all pending messages as failed
	p.inflight.Range(func(key, value interface{}) bool {
		pMsg := value.(*pendingMsg)
		pMsg.future.setError(err)
		p.inflight.Delete(key)
		p.inflightCount.Add(-1)
		return true
	})
}

// Compile-time interface checks
var _ OrderedPublisher = &orderedPublisher{}

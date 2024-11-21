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
	msg     *nats.Msg
	future  *pubFuture
	ctx     context.Context
	timeout time.Time
}

func NewOrderedPublisher(nc *nats.Conn, config OrderedPublisherConfig) (OrderedPublisher, error) {
	config.setDefaults()

	p := &orderedPublisher{
		publisher: &publisher{
			nc:     nc,
			config: *config.toPublisherConfig(),
		},
		config:    config,
		queue:     make(chan *pendingMsg, queueSize),
		shutdown:  make(chan struct{}),
		reset:     make(chan struct{}),
		resetDone: make(chan struct{}),
	}

	// Validate like jetstream does
	if err := p.validate(); err != nil {
		return nil, fmt.Errorf("invalid ordered publisher config: %w", err)
	}

	// Subscribe to responses
	p.inbox = nc.NewInbox()
	sub, err := nc.Subscribe(p.inbox+".*", p.handleResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to create response subscription: %w", err)
	}
	p.subscription = sub

	p.wg.Add(2)
	go p.publishLoop()
	go p.timeoutLoop()
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
func (p *orderedPublisher) PublishAsync(ctx context.Context, request PublishRequest) (PubFuture, error) {
	if p.pendingCount() >= p.config.MaxPending {
		return nil, fmt.Errorf("publisher max pending messages reached")
	}
	if err := p.validateRequest(request); err != nil {
		return nil, err
	}

	msg, err := p.createMsg(request)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	future := newPubFuture(msg)
	pMsg := &pendingMsg{
		msg:     msg,
		future:  future,
		ctx:     ctx,
		timeout: time.Now().Add(p.config.AckWait),
	}

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
	close(p.shutdown)
	if err := p.subscription.Unsubscribe(); err != nil {
		return fmt.Errorf("failed to unsubscribe: %w", err)
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
			now := time.Now()
			p.inflight.Range(func(key, value interface{}) bool {
				pMsg := value.(*pendingMsg)
				if now.After(pMsg.timeout) {
					pMsg.future.setError(errors.New("publish ack timeout"))
					p.inflight.Delete(key)
					p.inflightCount.Add(-1)
				}
				// TODO: Could implement retry logic here
				return true
			})
		case <-p.shutdown:
			return
		}
	}
}

// createMsg to override the publisher's create message and add reply subject
func (p *orderedPublisher) createMsg(request PublishRequest) (*nats.Msg, error) {
	msg, err := p.publisher.createMsg(request)
	if err != nil {
		return nil, err
	}

	// Generate unique reply subject
	msg.Reply = p.inbox + "." + uuid.NewString()
	return msg, nil
}

func (p *orderedPublisher) processMessage(pubMsg *pendingMsg) {
	// Check if the publish context is still valid
	if err := pubMsg.ctx.Err(); err != nil {
		pubMsg.future.setError(err)
		return
	}

	// check if the msg hasn't been cancelled
	if pubMsg.future.Err() != nil {
		return
	}

	// Publish with reply subject
	if err := p.nc.PublishMsg(pubMsg.msg); err != nil {
		pubMsg.future.setError(err)
		return
	}

	p.inflight.Store(pubMsg.msg.Reply, pubMsg)
	p.inflightCount.Add(1)
}

func (p *orderedPublisher) handleResponse(msg *nats.Msg) {
	// Look up pending future
	pMsg, ok := p.inflight.Load(msg.Subject)
	if !ok {
		// Response for unknown request - ignore after logging
		log.Debug().Str("subject", msg.Subject).Msg("received response for unknown request")
		return
	}
	defer func() {
		p.inflight.Delete(msg.Subject)
		p.inflightCount.Add(-1)
	}()

	future := pMsg.(*pendingMsg).future

	res, err := ParseResult(msg)
	if err != nil {
		future.setError(err)
		return
	}

	if res.Error != "" {
		// TODO: requeue with backoff, but keep track of retries
		future.setError(res.Err())
	} else {
		future.setResult(res)
	}
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

package libp2p

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	realsync "sync"

	sync "github.com/lukemarsden/golang-mutex-tracer"

	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const JobEventChannel = "bacalhau-job-event"

type LibP2PTransport struct {
	// Cleanup manager for resource teardown on exit:
	cm *system.CleanupManager

	subscribeFunctions   []transport.SubscribeFn
	mutex                sync.RWMutex
	host                 host.Host
	port                 int
	peers                []string
	pubSub               *pubsub.PubSub
	jobEventTopic        *pubsub.Topic
	jobEventSubscription *pubsub.Subscription
}

func NewTransport(cm *system.CleanupManager, port int, peers []string) (*LibP2PTransport, error) {
	usePeers := []string{}

	for _, p := range peers {
		if p != "" {
			usePeers = append(usePeers, p)
		}
	}

	prvKey, err := config.GetPrivateKey(fmt.Sprintf("private_key.%d", port))
	if err != nil {
		return nil, err
	}

	// 0.0.0.0 will listen on any interface device.
	sourceMultiAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))
	if err != nil {
		return nil, err
	}

	h, err := libp2p.New(
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(prvKey),
	)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	cm.RegisterCallback(func() error {
		cancel()
		return nil
	})

	ps, err := pubsub.NewGossipSub(ctx, h, pubsub.WithPeerExchange(true))
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	jobEventTopic, err := ps.Join(JobEventChannel)
	if err != nil {
		return nil, err
	}

	jobEventSubscription, err := jobEventTopic.Subscribe()
	if err != nil {
		return nil, err
	}

	libp2pTransport := &LibP2PTransport{
		cm:                   cm,
		subscribeFunctions:   []transport.SubscribeFn{},
		host:                 h,
		port:                 port,
		peers:                usePeers,
		pubSub:               ps,
		jobEventTopic:        jobEventTopic,
		jobEventSubscription: jobEventSubscription,
	}

	libp2pTransport.mutex.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "LibP2PTransport.mutex",
	})
	return libp2pTransport, nil
}

/*

  public api

*/

func (t *LibP2PTransport) HostID(ctx context.Context) (string, error) {
	return t.host.ID().String(), nil
}

func (t *LibP2PTransport) GetPeers(ctx context.Context) (map[string][]peer.ID, error) {
	response := map[string][]peer.ID{}
	for _, topic := range t.pubSub.GetTopics() {
		peers := t.pubSub.ListPeers(topic)
		response[topic] = peers
	}
	return response, nil
}

func (t *LibP2PTransport) Start(ctx context.Context) error {
	if len(t.subscribeFunctions) == 0 {
		panic("Programming error: no subscribe func, please call Subscribe immediately after constructing interface")
	}

	t.cm.RegisterCallback(func() error {
		return t.Shutdown(ctx)
	})

	err := t.connectToPeers(ctx)
	if err != nil {
		return err
	}

	go t.listenForEvents(ctx)

	log.Trace().Msg("Libp2p transport has started")

	return nil
}

func (t *LibP2PTransport) Shutdown(ctx context.Context) error {
	closeErr := t.host.Close()

	if closeErr != nil {
		log.Error().Msgf("Libp2p transport had error stopping: %s", closeErr.Error())
	} else {
		log.Debug().Msg("Libp2p transport has stopped")
	}

	return nil
}

func (t *LibP2PTransport) Publish(ctx context.Context, ev executor.JobEvent) error {
	return t.writeJobEvent(ctx, ev)
}

func (t *LibP2PTransport) Subscribe(fn transport.SubscribeFn) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.subscribeFunctions = append(t.subscribeFunctions, fn)
}

/*

  libp2p

*/

func (t *LibP2PTransport) connectToPeers(ctx context.Context) error {
	ctx, span := newSpan(ctx, "Connect")
	defer span.End()

	if len(t.peers) == 0 {
		return nil
	}

	for _, peerAddress := range t.peers {
		maddr, err := multiaddr.NewMultiaddr(peerAddress)
		if err != nil {
			return err
		}

		// Extract the peer ID from the multiaddr.
		info, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			return err
		}

		t.host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
		err = t.host.Connect(ctx, *info)
		if err != nil {
			return err
		}
		log.Trace().Msgf("Libp2p transport connected to: %s", peerAddress)
	}

	return nil
}

/*

  pub / sub

*/

// we wrap our events on the wire in this envelope so
// we can pass our tracing context to remote peers
type jobEventEnvelope struct {
	SentTime  time.Time              `json:"sent_time"`
	JobEvent  executor.JobEvent      `json:"job_event"`
	TraceData propagation.MapCarrier `json:"trace_data"`
}

func (t *LibP2PTransport) writeJobEvent(ctx context.Context, event executor.JobEvent) error {
	traceData := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, &traceData)

	bs, err := json.Marshal(jobEventEnvelope{
		JobEvent:  event,
		TraceData: traceData,
		SentTime:  time.Now(),
	})
	if err != nil {
		return err
	}

	log.Trace().Msgf("Sending event %s: %s", event.EventName.String(), string(bs))
	return t.jobEventTopic.Publish(ctx, bs)
}

func (t *LibP2PTransport) readMessage(msg *pubsub.Message) {
	// TODO: we would enforce the claims to SourceNodeID here
	// i.e. msg.ReceivedFrom() should match msg.Data.JobEvent.SourceNodeID
	payload := jobEventEnvelope{}
	err := json.Unmarshal(msg.Data, &payload)
	if err != nil {
		log.Error().Msgf("error unmarshalling libp2p event: %v", err)
		return
	}

	now := time.Now()
	then := payload.SentTime
	latency := now.Sub(then)
	latencyMilli := int64(latency / time.Millisecond)
	if latencyMilli > 500 { //nolint:gomnd
		log.Warn().Msgf(
			"[%s=>%s] VERY High message latency: %d ms (%s)",
			payload.JobEvent.SourceNodeID[:8],
			t.host.ID().String()[:8],
			latencyMilli, payload.JobEvent.EventName.String(),
		)
	} else if latencyMilli > 50 { //nolint:gomnd
		log.Warn().Msgf(
			"[%s=>%s] High message latency: %d ms (%s)",
			payload.JobEvent.SourceNodeID[:8],
			t.host.ID().String()[:8],
			latencyMilli, payload.JobEvent.EventName.String(),
		)
	} else {
		log.Trace().Msgf(
			"[%s=>%s] Message latency: %d ms (%s)",
			payload.JobEvent.SourceNodeID[:8],
			t.host.ID().String()[:8],
			latencyMilli, payload.JobEvent.EventName.String(),
		)
	}

	log.Trace().Msgf("Received event %s: %+v", payload.JobEvent.EventName.String(), payload)

	// Notify all the listeners in this process of the event:
	jobCtx := otel.GetTextMapPropagator().Extract(context.Background(), payload.TraceData)

	ev := payload.JobEvent
	// NOTE: Do not use msg.ReceivedFrom as the original sender, it's not. It's
	// the node which gossiped the message to us, which might be different.
	// (was: ev.SourceNodeID = msg.ReceivedFrom.String())

	var wg realsync.WaitGroup
	func() {
		t.mutex.RLock()
		defer t.mutex.RUnlock()

		for _, fn := range t.subscribeFunctions {
			wg.Add(1)
			go func(f transport.SubscribeFn) {
				defer wg.Done()
				f(jobCtx, ev)
			}(fn)
		}
	}()
	wg.Wait()
}

func (t *LibP2PTransport) listenForEvents(ctx context.Context) {
	for {
		msg, err := t.jobEventSubscription.Next(ctx)
		if err != nil {
			if err == context.Canceled || err == context.DeadlineExceeded {
				log.Trace().Msgf("libp2p transport shutting down: %v", err)
			} else {
				log.Error().Msgf(
					"libp2p encountered an unexpected error, shutting down: %v", err)
			}
			return
		}
		go t.readMessage(msg)
	}
}

func newSpan(ctx context.Context, apiName string) (context.Context, trace.Span) {
	return system.Span(ctx, "transport/libp2p", apiName)
}

// Compile-time interface check:
var _ transport.Transport = (*LibP2PTransport)(nil)

package libp2p

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

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
	ctx                  sync.Mutex
	host                 host.Host
	port                 int
	pubSub               *pubsub.PubSub
	jobEventTopic        *pubsub.Topic
	jobEventSubscription *pubsub.Subscription
}

func makeLibp2pHost(port int) (host.Host, error) {
	prvKey, err := config.GetPrivateKey(fmt.Sprintf("private_key.%d", port))
	if err != nil {
		return nil, err
	}

	// 0.0.0.0 will listen on any interface device.
	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))

	// libp2p.New constructs a new libp2p Host.
	// Other options can be added here.
	return libp2p.New(
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(prvKey),
	)
}

func NewTransport(cm *system.CleanupManager, port int) (*LibP2PTransport, error) {
	h, err := makeLibp2pHost(port)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	cm.RegisterCallback(func() error {
		cancel()
		return nil
	})

	ps, err := pubsub.NewGossipSub(ctx, h)
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
		pubSub:               ps,
		jobEventTopic:        jobEventTopic,
		jobEventSubscription: jobEventSubscription,
	}

	// // setup the event writer
	// libp2pTransport.genericTransport = transport.NewGenericTransport(
	// 	h.ID().String(),
	// 	func(ctx context.Context, event executor.JobEvent) error {
	// 		return libp2pTransport.writeJobEvent(ctx, event)
	// 	},
	// )

	return libp2pTransport, nil
}

/////////////////////////////////////////////////////////////
/// LIFECYCLE
/////////////////////////////////////////////////////////////

func (t *LibP2PTransport) HostID(ctx context.Context) (string, error) {
	return t.host.ID().String(), nil
}

func (t *LibP2PTransport) Start(ctx context.Context) error {
	if len(t.subscribeFunctions) == 0 {
		panic("Programming error: no subscribe func, please call Subscribe immediately after constructing interface")
	}

	log.Debug().Msg("Libp2p transport has started")

	t.cm.RegisterCallback(func() error {
		t.host.Close()
		log.Debug().Msg("Libp2p transport has stopped")

		return t.Shutdown(ctx)
	})

	log.Debug().Msg("libp2p transport is starting...")
	go t.readLoopJobEvents(ctx)

	return nil
}

func (t *LibP2PTransport) Shutdown(ctx context.Context) error {
	return t.genericTransport.Shutdown(ctx)
}

/////////////////////////////////////////////////////////////
/// READ OPERATIONS
/////////////////////////////////////////////////////////////

func (t *Transport) List(ctx context.Context) (transport.ListResponse, error) {
	ctx, span := newSpan(ctx, "List")
	defer span.End()

	return t.genericTransport.List(ctx)
}

func (t *Transport) Get(ctx context.Context, id string) (executor.Job, error) {
	ctx, span := newSpan(ctx, "Get")
	defer span.End()

	return t.genericTransport.Get(ctx, id)
}

func (t *Transport) Subscribe(ctx context.Context, fn transport.SubscribeFn) {
	ctx, span := newSpan(ctx, "Subscribe")
	defer span.End()

	t.genericTransport.Subscribe(ctx, fn)
}

/////////////////////////////////////////////////////////////
/// INTERNAL IMPLEMENTATION
/////////////////////////////////////////////////////////////

func (t *Transport) Connect(ctx context.Context, peerConnect string) error {
	ctx, span := newSpan(ctx, "Connect")
	defer span.End()

	if peerConnect == "" {
		return nil
	}

	maddr, err := multiaddr.NewMultiaddr(peerConnect)
	if err != nil {
		return err
	}

	// Extract the peer ID from the multiaddr.
	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return err
	}

	t.Host.Peerstore().AddAddrs(
		info.ID, info.Addrs, peerstore.PermanentAddrTTL)

	return t.Host.Connect(ctx, *info)
}

type jobEventData struct {
	JobEvent  executor.JobEvent      `json:"job_event"`
	TraceData propagation.MapCarrier `json:"trace_data"`
}

func (t *Transport) writeJobEvent(ctx context.Context, event executor.JobEvent) error {
	traceData := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, &traceData)

	bs, err := json.Marshal(jobEventData{
		JobEvent:  event,
		TraceData: traceData,
	})
	if err != nil {
		return err
	}

	log.Debug().Msgf("Sending event: %s", string(bs))
	return t.JobEventTopic.Publish(ctx, bs)
}

func (t *Transport) readLoopJobEvents(ctx context.Context) {
	for {
		msg, err := t.JobEventSubscription.Next(ctx)
		if err != nil {
			if err == context.Canceled || err == context.DeadlineExceeded {
				log.Info().Msgf("libp2p transport shutting down: %v", err)
			} else {
				log.Error().Msgf(
					"libp2p encountered an unexpected error, shutting down: %v", err)
			}

			return
		}

		jed := jobEventData{}
		if err = json.Unmarshal(msg.Data, &jed); err != nil {
			log.Error().Msgf("error unmarshalling libp2p event: %v", err)
			continue
		}
		log.Trace().Msgf("Received event: %+v", jed)

		// Notify all the listeners in this process of the event:
		ctx = otel.GetTextMapPropagator().Extract(ctx, jed.TraceData)
		t.genericTransport.ReadEvent(ctx, jed.JobEvent)
	}
}

func newSpan(ctx context.Context, apiName string) (context.Context, trace.Span) {
	return system.Span(ctx, "transport/libp2p", apiName)
}

func (t *LibP2PTransport) Subscribe(fn transport.SubscribeFn) {
	t.ctx.Lock()
	defer t.ctx.Unlock()
	t.subscribeFunctions = append(t.subscribeFunctions, fn)
}

// Compile-time interface check:
var _ transport.Transport = (*LibP2PTransport)(nil)

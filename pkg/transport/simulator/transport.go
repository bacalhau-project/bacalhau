package simulator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	realsync "sync"

	sync "github.com/lukemarsden/golang-mutex-tracer"

	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport"
	"github.com/gorilla/websocket"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/propagation"
)

type SimulatorTransport struct {
	cm                 *system.CleanupManager
	id                 string
	url                string
	subscribeFunctions []transport.SubscribeFn
	websocket          *websocket.Conn
	websocketMutex     sync.Mutex
	privateKey         crypto.PrivKey
	subscriptionMutex  sync.RWMutex
}

func NewTransport(
	ctx context.Context,
	cm *system.CleanupManager,
	id string,
	// this should be scheme://host:port and not contain a path
	url string,
) (*SimulatorTransport, error) {
	prvKey, err := config.GetPrivateKey(fmt.Sprintf("private_key.%s", id))
	if err != nil {
		return nil, err
	}

	return &SimulatorTransport{
		cm:                 cm,
		id:                 id,
		url:                url,
		subscribeFunctions: []transport.SubscribeFn{},
		privateKey:         prvKey,
	}, nil
}

/*

  public api

*/

func (t *SimulatorTransport) HostID() string {
	return t.id
}

func (t *SimulatorTransport) HostAddrs() ([]multiaddr.Multiaddr, error) {
	return []multiaddr.Multiaddr{}, nil
}

func (t *SimulatorTransport) Start(ctx context.Context) error {
	conn, _, err := websocket.DefaultDialer.Dial(t.url, nil)
	if err != nil {
		return err
	}
	t.websocket = conn
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Error().Msgf(
					"Simulation Transport error reading message %s", err.Error())
				continue
			}

			payload := jobEventEnvelope{}
			err = json.Unmarshal(msg, &payload)
			if err != nil {
				log.Error().Msgf(
					"Simulation Transport error unmarshalling message %s", err.Error())
				continue
			}

			go t.readMessage(&payload)
		}
	}()
	return nil
}

func (t *SimulatorTransport) Shutdown(ctx context.Context) error {
	return t.websocket.Close()
}

func (t *SimulatorTransport) Publish(ctx context.Context, ev model.JobEvent) error {
	return t.writeJobEvent(ctx, ev)
}

func (t *SimulatorTransport) PublishNode(ctx context.Context, ev model.NodeEvent) error {
	return nil
}

func (t *SimulatorTransport) Subscribe(ctx context.Context, fn transport.SubscribeFn) {
	t.subscriptionMutex.Lock()
	defer t.subscriptionMutex.Unlock()
	t.subscribeFunctions = append(t.subscribeFunctions, fn)
}

func (t *SimulatorTransport) SubscribeNode(ctx context.Context, fn transport.SubscribeFnNode) {

}

func (t *SimulatorTransport) Encrypt(ctx context.Context, data, keyBytes []byte) ([]byte, error) {
	// XXX Simulator doesn't implement encryption right now. In the future, to
	// support more sophisticated bad actors, it will need to.
	return data, nil
}

func (t *SimulatorTransport) Decrypt(ctx context.Context, data []byte) ([]byte, error) {
	return data, nil
}

/*

  pub / sub

*/

// we wrap our events on the wire in this envelope so
// we can pass our tracing context to remote peers
type jobEventEnvelope struct {
	SentTime  time.Time              `json:"sent_time"`
	JobEvent  model.JobEvent         `json:"job_event"`
	TraceData propagation.MapCarrier `json:"trace_data"`
}

func (t *SimulatorTransport) writeJobEvent(ctx context.Context, event model.JobEvent) error {
	t.websocketMutex.Lock()
	defer t.websocketMutex.Unlock()

	publicKeyBytes, err := t.privateKey.GetPublic().Raw()
	if err != nil {
		return err
	}
	event.SenderPublicKey = publicKeyBytes
	bs, err := json.Marshal(jobEventEnvelope{
		JobEvent:  event,
		TraceData: map[string]string{},
		SentTime:  time.Now(),
	})
	if err != nil {
		return err
	}

	if t.websocket == nil {
		return fmt.Errorf("websocket not connected")
	}

	log.Debug().Msgf("Sending event %s: %s", event.EventName.String(), string(bs))
	err = t.websocket.WriteMessage(websocket.TextMessage, bs)
	if err != nil {
		return err
	}
	return nil
}

func (t *SimulatorTransport) readMessage(payload *jobEventEnvelope) {
	now := time.Now()
	then := payload.SentTime
	latency := now.Sub(then)
	latencyMilli := int64(latency / time.Millisecond)
	if latencyMilli > 500 { //nolint:gomnd
		log.Warn().Msgf(
			"[%s=>%s] VERY High message latency: %d ms (%s)",
			payload.JobEvent.SourceNodeID[:8],
			t.id[:8],
			latencyMilli, payload.JobEvent.EventName.String(),
		)
	} else if latencyMilli > 50 { //nolint:gomnd
		log.Warn().Msgf(
			"[%s=>%s] High message latency: %d ms (%s)",
			payload.JobEvent.SourceNodeID[:8],
			t.id[:8],
			latencyMilli, payload.JobEvent.EventName.String(),
		)
	} else {
		log.Trace().Msgf(
			"[%s=>%s] Message latency: %d ms (%s)",
			payload.JobEvent.SourceNodeID[:8],
			t.id[:8],
			latencyMilli, payload.JobEvent.EventName.String(),
		)
	}

	log.Trace().Msgf("Received event %s: %+v", payload.JobEvent.EventName.String(), payload)
	ev := payload.JobEvent

	var wg realsync.WaitGroup
	func() {
		t.subscriptionMutex.RLock()
		defer t.subscriptionMutex.RUnlock()

		for _, fn := range t.subscribeFunctions {
			wg.Add(1)
			go func(f transport.SubscribeFn) {
				defer wg.Done()
				err := f(context.Background(), ev)
				if err != nil {
					log.Error().Msgf("error in handle event: %s\n%+v", err, ev)
				}
			}(fn)
		}
	}()
	wg.Wait()
}

// Compile-time interface check:
var _ transport.Transport = (*SimulatorTransport)(nil)

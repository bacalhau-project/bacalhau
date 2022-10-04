package simulator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/eventhandler"
	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/propagation"
)

type jobEventEnvelope struct {
	SentTime  time.Time              `json:"sent_time"`
	JobEvent  model.JobEvent         `json:"job_event"`
	TraceData propagation.MapCarrier `json:"trace_data"`
}

type SimulationAPIServer struct {
	Host          string
	Port          int
	localDB       localdb.LocalDB
	eventConsumer *eventhandler.ChainedJobEventHandler
}

const ServerReadHeaderTimeout = 10 * time.Second

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
var connections = []*websocket.Conn{}

// NewServer returns a new API server for a requester node.
func NewServer(
	ctx context.Context,
	host string,
	port int,
	localDB localdb.LocalDB,
) *SimulationAPIServer {
	eventConsumer := eventhandler.NewChainedJobEventHandler(system.NewNoopContextProvider())
	eventConsumer.AddHandlers(
		localdb.NewLocalDBEventHandler(localDB),
	)
	server := &SimulationAPIServer{
		Host:          host,
		Port:          port,
		localDB:       localDB,
		eventConsumer: eventConsumer,
	}
	return server
}

// GetURI returns the HTTP URI that the server is listening on.
func (apiServer *SimulationAPIServer) GetURI() string {
	return fmt.Sprintf("%s:%d", apiServer.Host, apiServer.Port)
}

// ListenAndServe listens for and serves HTTP requests against the API server.
func (apiServer *SimulationAPIServer) ListenAndServe(ctx context.Context, cm *system.CleanupManager) error {
	sm := http.NewServeMux()
	sm.HandleFunc("/websocket", apiServer.websocketHandler)

	srv := http.Server{
		Handler:           sm,
		Addr:              fmt.Sprintf("%s:%d", apiServer.Host, apiServer.Port),
		ReadHeaderTimeout: ServerReadHeaderTimeout,
	}

	log.Debug().Msgf("Simulation API server listening on %s...", apiServer.GetURI())

	cm.RegisterCallback(func() error {
		return srv.Shutdown(ctx)
	})

	err := srv.ListenAndServe()
	if err == http.ErrServerClosed {
		log.Debug().Msgf(
			"API server closed on %s.", srv.Addr)
		return nil // expected error if the server is shut down
	}

	return err
}

func (apiServer *SimulationAPIServer) websocketHandler(res http.ResponseWriter, req *http.Request) {
	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Debug().Msgf("New websocket connection.")
	connections = append(connections, conn)

	for {
		_, message, err := conn.ReadMessage()

		if err != nil {
			log.Error().Msgf("error reading websocket message: %s\n", err.Error())
			break
		}

		payload := jobEventEnvelope{}
		err = json.Unmarshal(message, &payload)

		err = apiServer.eventConsumer.HandleJobEvent(context.Background(), payload.JobEvent)
		if err != nil {
			log.Error().Msgf("error writing job event to consumer: %s\n", err.Error())
			continue
		}

		// step 1: handle message i.e. mutate our internal state
		// switch event.EventName {
		// case model.JobEventBid:
		// 	fmt.Printf("received bid event: %s", event.JobID)
		// case model.JobEventBidAccepted:
		// 	fmt.Printf("received bid accepted event: %s", event.JobID)
		// }

		// step 2: broacast the message back to all subscribers
		for _, conn := range connections {
			err := conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Error().Msgf("error writing event JSON: %s\n", err.Error())
			}
		}
	}
}

package simulator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type SimulationAPIServer struct {
	Host string
	Port int
}

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
) *SimulationAPIServer {
	server := &SimulationAPIServer{
		Host: host,
		Port: port,
	}
	return server
}

// GetURI returns the HTTP URI that the server is listening on.
func (apiServer *SimulationAPIServer) GetURI() string {
	return fmt.Sprintf("%s:%d", apiServer.Host, apiServer.Port)
}

// ListenAndServe listens for and serves HTTP requests against the API server.
func (apiServer *SimulationAPIServer) ListenAndServe(ctx context.Context, cm *system.CleanupManager) error {
	http.HandleFunc("/websocket", apiServer.websocketHandler)

	log.Debug().Msgf("Simulation API server listening on %s...", apiServer.GetURI())

	err := http.ListenAndServe(fmt.Sprintf("%s:%d", apiServer.Host, apiServer.Port), nil)
	if err == http.ErrServerClosed {
		log.Debug().Msgf(
			"API server closed on %s.", apiServer.GetURI())
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

		var event model.JobEvent

		err = json.NewDecoder(bytes.NewReader(message)).Decode(&event)
		if err != nil {
			log.Error().Msgf("error parsing event JSON: %s\n", err.Error())
			break
		}

		log.Debug().Msgf("event: %+v\n", event)

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

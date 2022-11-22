package publicapi

import (
	"context"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// TODO: Godoc
func (apiServer *APIServer) websockets(res http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Debug().Msgf("New websocket connection.")
	defer conn.Close()

	// NB: jobId == "" is the case for subscriptions to "all events"
	jobId := req.URL.Query().Get("job_id")

	apiServer.WebsocketsMutex.Lock()
	defer apiServer.WebsocketsMutex.Unlock()

	sockets, ok := apiServer.Websockets[jobId]
	if !ok {
		sockets = []*websocket.Conn{}
		apiServer.Websockets[jobId] = sockets
	}
	apiServer.Websockets[jobId] = append(sockets, conn)

	if jobId != "" {
		// list events for job out of localDB and send them to the client
		events, err := apiServer.localdb.GetJobEvents(context.Background(), jobId)
		if err != nil {
			log.Error().Msgf("error listing job events: %s\n", err.Error())
			return
		}
		for _, event := range events {
			err := conn.WriteJSON(event)
			if err != nil {
				log.Error().Msgf("error writing event JSON: %s\n", err.Error())
			}
		}
	}

}

func (apiServer *APIServer) HandleJobEvent(ctx context.Context, event model.JobEvent) (err error) {

	apiServer.WebsocketsMutex.RLock()
	defer apiServer.WebsocketsMutex.RUnlock()

	dispatchAndCleanup := func(jobId string) {
		connections, ok := apiServer.Websockets[jobId]
		if !ok {
			return
		}
		errIdxs := []int{}
		for idx, connection := range connections {
			log.Debug().Msgf("sending %+v to %s/%d", event, jobId, idx)
			err := connection.WriteJSON(event)
			if err != nil {
				errIdxs = append(errIdxs, idx)
			}
		}
		// reverse errIdxs and clean up dud indexes
		for i := len(errIdxs)/2 - 1; i >= 0; i-- {
			opp := len(errIdxs) - 1 - i
			errIdxs[i], errIdxs[opp] = errIdxs[opp], errIdxs[i]
		}
		for _, idx := range errIdxs {
			connections = append(connections[:idx], connections[idx+1:]...)
		}
		apiServer.Websockets[jobId] = connections
	}
	dispatchAndCleanup("")
	dispatchAndCleanup(event.JobID)
	return nil
}

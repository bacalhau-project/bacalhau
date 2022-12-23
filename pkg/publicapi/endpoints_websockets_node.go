package publicapi

import (
	"context"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
)

// TODO: Godoc
func (apiServer *APIServer) websocketNode(res http.ResponseWriter, req *http.Request) {
	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Debug().Msgf("New websocket connection.")
	defer conn.Close()

	func() {
		apiServer.NodeWebsocketsMutex.Lock()
		defer apiServer.NodeWebsocketsMutex.Unlock()
		apiServer.NodeWebsockets = append(apiServer.NodeWebsockets, conn)
	}()

	for {
		// read and throw away any incoming messages, exit when client
		// disconnects (which is a sort of error)
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (apiServer *APIServer) HandleNodeEvent(ctx context.Context, event model.NodeEvent) (err error) {
	apiServer.NodeWebsocketsMutex.Lock()
	defer apiServer.NodeWebsocketsMutex.Unlock()

	errIdxs := []int{}
	for idx, connection := range apiServer.NodeWebsockets {
		// TODO: dispatch to subscribers in parallel, to avoid one slow
		// reader slowing all the others down.
		err := connection.WriteJSON(event)
		if err != nil {
			log.Error().Msgf(
				"error writing event to subscriber %d: %s, closing ws\n",
				idx, err.Error(),
			)
			errIdxs = append(errIdxs, idx)
			// close the connection, if possible, to allow the other side to
			// retry. Ignore errors from closing, since we are going to
			// delete this connection anyway.
			connection.Close()
		}
	}
	// reverse errIdxs (so we don't mess up the indexes for cleanup) and
	// clean up dud connections
	for i := len(errIdxs)/2 - 1; i >= 0; i-- {
		opp := len(errIdxs) - 1 - i
		errIdxs[i], errIdxs[opp] = errIdxs[opp], errIdxs[i]
	}
	for _, idx := range errIdxs {
		apiServer.NodeWebsockets = append(apiServer.NodeWebsockets[:idx], apiServer.NodeWebsockets[idx+1:]...)
	}

	return nil
}

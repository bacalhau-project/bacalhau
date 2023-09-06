package requester

import (
	"context"
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels/legacymodels"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// TODO: Godoc

func (s *Endpoint) websocketJobEvents(c echo.Context) error {
	ctx := c.Request().Context()
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	log.Ctx(ctx).Debug().Msgf("New websocketJobEvents connection.")
	defer conn.Close()

	// NB: jobId == "" is the case for subscriptions to "all events"

	// get job_id from query string
	jobID := c.QueryParam("job_id")

	func() {
		s.websocketsMutex.Lock()
		defer s.websocketsMutex.Unlock()

		sockets, ok := s.websockets[jobID]
		if !ok {
			sockets = []*websocket.Conn{}
			s.websockets[jobID] = sockets
		}
		s.websockets[jobID] = append(sockets, conn)
	}()

	if jobID != "" {
		// list events for job out of localDB and send them to the client
		events, err := s.jobStore.GetJobHistory(context.Background(), jobID, legacymodels.EventFilterOptions{})
		if err != nil {
			log.Ctx(ctx).Error().Msgf("error listing job events: %s\n", err.Error())
			return err
		}
		for _, event := range events {
			err := conn.WriteJSON(event)
			if err != nil {
				log.Ctx(ctx).Error().Msgf("error writing event JSON: %s\n", err.Error())
			}
		}
	}

	for {
		// read and throw away any incoming messages, exit when client
		// disconnects (which is a sort of error)
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}

	return nil
}

func (s *Endpoint) HandleJobEvent(ctx context.Context, event model.JobEvent) error {
	s.websocketsMutex.Lock()
	defer s.websocketsMutex.Unlock()

	dispatchAndCleanup := func(jobId string) {
		connections, ok := s.websockets[jobId]
		if !ok {
			return
		}
		errIdxs := []int{}
		for idx, connection := range connections {
			// TODO: dispatch to subscribers in parallel, to avoid one slow
			// reader slowing all the others down.
			err := connection.WriteJSON(event)
			if err != nil {
				log.Ctx(ctx).Error().Msgf(
					"error writing event to subscriber '%s'/%d: %s, closing ws\n",
					jobId, idx, err.Error(),
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
			connections = append(connections[:idx], connections[idx+1:]...)
		}
		s.websockets[jobId] = connections
	}
	dispatchAndCleanup("")
	dispatchAndCleanup(event.JobID)
	return nil
}

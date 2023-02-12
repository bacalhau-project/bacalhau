package model

import (
	"context"
	"fmt"

	bacalhau_model "github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/routing"
	"github.com/rs/zerolog/log"
)

type nodeEventHandler struct {
	eventChan chan bacalhau_model.NodeInfo
	firehose  *EventFirehose[bacalhau_model.NodeInfo]
	nodeDB    routing.NodeInfoStore
}

func newNodeEventHandler(
	host string,
	port int,
	nodeDB routing.NodeInfoStore,
) (*nodeEventHandler, error) {
	eventChan := make(chan bacalhau_model.NodeInfo)
	url := fmt.Sprintf("ws://%s:%d/requester/node/websocket", host, port)
	firehose := NewEventFirehose(url, eventChan)
	eventHandler := &nodeEventHandler{
		eventChan: eventChan,
		firehose:  firehose,
		nodeDB:    nodeDB,
	}
	return eventHandler, nil
}

func (handler *nodeEventHandler) start(ctx context.Context) {
	go handler.firehose.Start(ctx)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-handler.eventChan:
				err := handler.nodeDB.Add(ctx, ev)
				if err != nil {
					log.Info().Err(err).Msgf("failed to add node info to store: %+v", ev)
				}
			}
		}
	}()
}

package model

import (
	"context"
	"fmt"

	bacalhau_model "github.com/filecoin-project/bacalhau/pkg/model"
)

type nodeEventHandler struct {
	eventChan chan bacalhau_model.NodeEvent
	firehose  *EventFirehose[bacalhau_model.NodeEvent]
	nodeDB    *nodeDB
}

func newNodeEventHandler(
	host string,
	port int,
	nodeDB *nodeDB,
) (*nodeEventHandler, error) {
	eventChan := make(chan bacalhau_model.NodeEvent)
	url := fmt.Sprintf("ws://%s:%d/node/websocket", host, port)
	firehose := NewEventFirehose(url, eventChan)
	eventHandler := &nodeEventHandler{
		eventChan: eventChan,
		firehose:  firehose,
		nodeDB:    nodeDB,
	}
	return eventHandler, nil
}

func (handler *nodeEventHandler) start(ctx context.Context) {
	go handler.nodeDB.cleanupLoop(ctx)
	go handler.firehose.Start(ctx)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-handler.eventChan:
				handler.nodeDB.addEvent(ev)
			}
		}
	}()
}

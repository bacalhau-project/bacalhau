package model

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/localdb"
	bacalhau_model "github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
)

type jobEventBuffer struct {
	created time.Time
	exists  bool
	ignore  bool
	events  []bacalhau_model.JobEvent
}

type jobEventHandler struct {
	eventChan    chan bacalhau_model.JobEvent
	firehose     *EventFirehose[bacalhau_model.JobEvent]
	localDB      localdb.LocalDB
	eventHandler *localdb.LocalDBEventHandler
	eventBuffers map[string]*jobEventBuffer
	eventMutex   sync.Mutex
}

func newJobEventHandler(
	host string,
	port int,
	localDB localdb.LocalDB,
) (*jobEventHandler, error) {
	eventChan := make(chan bacalhau_model.JobEvent)
	url := fmt.Sprintf("ws://%s:%d/requester/websocket", host, port)
	firehose := NewEventFirehose(url, eventChan)
	eventHandler := &jobEventHandler{
		eventChan:    eventChan,
		firehose:     firehose,
		localDB:      localDB,
		eventHandler: localdb.NewLocalDBEventHandler(localDB),
		eventBuffers: map[string]*jobEventBuffer{},
	}
	return eventHandler, nil
}

func (handler *jobEventHandler) start(ctx context.Context) {
	// reap the event buffer so we don't accumulate memory forever
	ticker := time.NewTicker(1 * time.Minute)
	go handler.firehose.Start(ctx)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				handler.cleanEventBuffer()
			case ev := <-handler.eventChan:
				handler.readEvent(ctx, ev)
			}
		}
	}()
}

func (handler *jobEventHandler) writeEventToDatabase(ctx context.Context, event bacalhau_model.JobEvent) error {
	return handler.eventHandler.HandleJobEvent(ctx, event)
}

// sometimes events can be out of order and we need the job to exist
// before we record events against the job - it's OK if we hear about
// out of order events once the job exists in db (they have timestamps)
func (handler *jobEventHandler) readEvent(ctx context.Context, event bacalhau_model.JobEvent) {
	handler.eventMutex.Lock()
	defer handler.eventMutex.Unlock()
	eventBuffer, ok := handler.eventBuffers[event.JobID]

	// so this is the first event we have seen for this job
	// let's create a buffer for it
	if !ok {
		eventBuffer = &jobEventBuffer{
			created: time.Now(),
			exists:  false,
			ignore:  false,
			events:  []bacalhau_model.JobEvent{},
		}
		handler.eventBuffers[event.JobID] = eventBuffer
	}

	if event.EventName == bacalhau_model.JobEventCreated {
		isCanary := false
		for _, label := range event.Spec.Annotations {
			if label == "canary" {
				isCanary = true
				break
			}
		}
		for _, entrypointPart := range event.Spec.Docker.Entrypoint {
			if entrypointPart == "hello λ!" {
				isCanary = true
				break
			}
		}
		if isCanary {
			eventBuffer.ignore = true
			return
		}
		eventBuffer.exists = true
		err := handler.writeEventToDatabase(ctx, event)
		if err != nil {
			log.Error().Msgf("error writing event to database: %s", err.Error())
		}
		for _, bufferedEvent := range eventBuffer.events {
			err := handler.writeEventToDatabase(ctx, bufferedEvent)
			if err != nil {
				log.Error().Msgf("error writing event to database: %s", err.Error())
			}
		}
	} else if !eventBuffer.exists {
		eventBuffer.events = append(eventBuffer.events, event)
	} else {
		err := handler.writeEventToDatabase(ctx, event)
		if err != nil {
			log.Error().Msgf("error writing event to database: %s", err.Error())
		}
	}
}

func (handler *jobEventHandler) cleanEventBuffer() {
	handler.eventMutex.Lock()
	defer handler.eventMutex.Unlock()
	// clean up all event buffers that are older than 1 minute
	// if there is a 1 minute gap between hearing the first out of order
	// event and then hearing the create event then something has
	// gone badly wrong - this should be more like < 100ms in reality
	for jobID, eventBuffer := range handler.eventBuffers {
		if time.Since(eventBuffer.created) > 1*time.Minute {
			delete(handler.eventBuffers, jobID)
		}
	}
}

package model

import (
	"context"
	"encoding/json"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type firehoseEvent interface {
	model.JobEvent | model.NodeEvent
}

type EventFirehose[T firehoseEvent] struct {
	url        string
	active     bool
	connection *websocket.Conn
	eventChan  chan T
}

func NewEventFirehose[T firehoseEvent](url string, eventChan chan T) *EventFirehose[T] {
	return &EventFirehose[T]{
		url:       url,
		active:    true,
		eventChan: eventChan,
	}
}

func (firehose *EventFirehose[T]) connectLoop() {
	for {
		err := firehose.connect()
		if err != nil {
			log.Debug().Msgf("Websocket connection error %s", err.Error())
		}
		time.Sleep(time.Second * 1)
		if !firehose.active {
			break
		}
	}
}

func (firehose *EventFirehose[T]) keepAliveLoop() {
	for {
		if firehose.connection != nil {
			err := firehose.connection.WriteMessage(websocket.PingMessage, []byte("ping"))
			if err != nil {
				log.Debug().Msgf("Keepalive error %s", err.Error())
			}
		}
		time.Sleep(time.Second * 1)
		if !firehose.active {
			break
		}
	}
}

func (firehose *EventFirehose[T]) connect() error {
	connection, _, err := websocket.DefaultDialer.Dial(firehose.url, nil)
	if err != nil {
		return err
	}
	firehose.connection = connection
	log.Debug().Msgf("websocket connected %s", firehose.url)
	for {
		_, envelopeBytes, err := connection.ReadMessage()
		if err != nil {
			log.Debug().Msgf("websocket connection error: %s", err.Error())
			break
		}
		var event T
		err = json.Unmarshal(envelopeBytes, &event)
		if err != nil {
			log.Debug().Msgf("websocket json parse error: '%s' - %s", string(envelopeBytes), err.Error())
			continue
		}
		firehose.eventChan <- event
	}
	connection.Close()
	return nil
}

func (firehose *EventFirehose[T]) Start(ctx context.Context) {
	go firehose.connectLoop()
	go firehose.keepAliveLoop()
	<-ctx.Done()
	firehose.active = false
	if firehose.connection != nil {
		firehose.connection.Close()
	}
}

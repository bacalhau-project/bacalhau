package eventhandler

import (
	"context"
	"fmt"
	"io/fs"
	"os"

	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Tracer is a JobEventHandler that will marshal the received event to a
// file-based log.
//
// Note that we don't need any mutexes here because writing to an os.File is
// thread-safe (see https://github.com/rs/zerolog/blob/master/writer.go#L33)
type Tracer struct {
	LogFile *os.File
	Logger  zerolog.Logger
}

const eventTracerFilePerms fs.FileMode = 0644

// Returns an eventhandler.Tracer that writes to config.GetEventTracerPath(), or
// an error if the file can't be opened.
func NewTracer() (*Tracer, error) {
	return NewTracerToFile(config.GetEventTracerPath())
}

// Returns an eventhandler.Tracer that writes to the specified filename, or an
// error if the file can't be opened.
func NewTracerToFile(filename string) (*Tracer, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, eventTracerFilePerms)
	if err != nil {
		return nil, err
	}

	return &Tracer{
		LogFile: file,
		Logger:  zerolog.New(file).With().Timestamp().Logger(),
	}, nil
}

// Returns an eventhandler.Tracer that uses zerolog configured default output (e.g. stdout)
func NewDefaultTracer() *Tracer {
	return &Tracer{
		Logger: log.With().Logger(),
	}
}

// HandleJobEvent implements JobEventHandler
func (t *Tracer) HandleJobEvent(ctx context.Context, event model.JobEvent) error {
	trace(t.Logger, event)
	return nil
}

func trace[Event any](log zerolog.Logger, event Event) {
	log.Log().
		Str("Type", fmt.Sprintf("%T", event)).
		Func(func(e *zerolog.Event) {
			// TODO: #828 Potential hotspot - marshaling is expensive, and
			// we do it for every event.
			eventJSON, err := model.JSONMarshalWithMax(event)
			if err == nil {
				e.RawJSON("Event", eventJSON)
			} else {
				e.AnErr("MarshalError", err)
			}
		}).Send()
}

func (t *Tracer) Shutdown() error {
	if t.LogFile != nil {
		err := t.LogFile.Close()
		t.LogFile = nil
		t.Logger = zerolog.Nop()
		return err
	}
	return nil
}

var _ JobEventHandler = (*Tracer)(nil)

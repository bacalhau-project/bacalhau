package wasmlogs

import (
	"context"
	"io"
	"sync"

	"github.com/rs/zerolog/log"
)

const (
	defaultBufSize = 16 * 1024
)

// MultiReaderLogger is derived from moby/moby Copier and shares the
// same pattern of use where multiple readers are written to a single
// consumer that will quickly (and asynchronously) enqueue a message
// for processing. This allows our wasm runtime to provide one half
// of a pipe to this struct, and the write half to the wasm instance
// which will expose it to the compute. The MultiReaderLogger will
// consume data from each reader until it sends an EOF (or error).
type MultiReaderLogger struct {
	ctx       context.Context
	readers   map[string]io.Reader
	log       LogMediator
	wg        sync.WaitGroup
	closed    chan bool
	closeOnce sync.Once
}

// NewMultiReaderLogger creates a new MultiReaderLogger which will read from the
// supplied readers before writing them to a sink.  The name associated with each
// reader is assumed to be the text that describes the source of the data, for
// instance `stdout` or `stderr`.
func NewMultiReaderLogger(readers map[string]io.Reader, mediator LogMediator) *MultiReaderLogger {
	return &MultiReaderLogger{
		ctx:     context.Background(),
		readers: readers,
		log:     mediator,
	}
}

func (m *MultiReaderLogger) Copy() {
	for name, reader := range m.readers {
		m.wg.Add(1)
		go m.copy(name, reader)
	}

	// Blocks/Waits for all of the readers to complete
	m.wg.Wait()
}

// copy, intended to be run in a goroutine, will read until complete and
// enqueue messages, then when we get an error or EOF, tell the WaitGroup
// we're done
func (m *MultiReaderLogger) copy(name string, reader io.Reader) {
	buffer := make([]byte, defaultBufSize)

	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			msg := NewMessage(name, buffer[0:n])

			// Enqueue the message
			logErr := m.log.WriteLog(msg)
			if logErr != nil {
				log.Ctx(m.ctx).Err(err).Msgf("failed to write to mediator's %s", name)
				break
			}
		}

		if err != nil {
			if err != io.EOF {
				log.Ctx(m.ctx).Err(err).Msgf("failed to read from %s wasmlog stream", name)
			}
			break
		}
	}
	m.wg.Done()
}

func (m *MultiReaderLogger) Close() {
	m.closeOnce.Do(func() {
		close(m.closed)
	})
}

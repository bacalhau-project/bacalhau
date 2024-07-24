package util

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

const DefaultRefreshInterval = time.Millisecond
const ESC = 27

// clear the line and move the cursor up
var clear = fmt.Sprintf("%c[%dA%c[2K", ESC, 1, ESC)

// LiveTableWriter is a buffered table writer that updates the terminal.
// The contents of LiveTableWriter will be flushed on a timed interval or
// when Flush is called.
type LiveTableWriter struct {
	// Out is the writer to write the table to.
	Out io.Writer

	// RefreshInterval is the time the UI should refresh
	RefreshInterval time.Duration

	ticker *time.Ticker
	tdone  chan bool

	buf      bytes.Buffer
	mu       *sync.Mutex
	rowCount int
}

// NewLiveTableWriter returns a new LiveTableWriter with defaults
func NewLiveTableWriter() *LiveTableWriter {
	return &LiveTableWriter{
		Out:             io.Writer(os.Stdout),
		RefreshInterval: DefaultRefreshInterval,
		mu:              &sync.Mutex{},
	}
}

// Flush writes out the table and resets the buffer. It should be called after
// the last call to Write to ensure that any data buffered in the Writer is
// written to the output. An error is returned if the contents of the buffer
// cannot be written to the underlying output stream.
func (w *LiveTableWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Do Nothing if the Buffer is Empty
	if len(w.buf.Bytes()) == 0 {
		return nil
	}

	w.clearRows()

	rows := 0
	for _, b := range w.buf.Bytes() {
		if b == '\n' {
			rows++
		}
	}
	w.rowCount = rows
	_, err := w.Out.Write(w.buf.Bytes())
	w.buf.Reset()
	return err
}

// Start starts the listener in a non-blocking manner.
func (w *LiveTableWriter) Start() {
	if w.ticker == nil {
		w.ticker = time.NewTicker(w.RefreshInterval)
		w.tdone = make(chan bool)
	}

	go w.Listen()
}

// Write saves the table contents of buf to the writer b.
// The only errors are ones encountered while writing to the underlying buffer.
func (w *LiveTableWriter) Write(buf []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(buf)
}

// Listen listens for updates to live table writer's buffer and flushes to output
// provided.
func (w *LiveTableWriter) Listen() {
	for {
		select {
		case <-w.ticker.C:
			if w.ticker != nil {
				_ = w.Flush()
			}
		case <-w.tdone:
			w.mu.Lock()
			w.ticker.Stop()
			w.ticker = nil
			w.mu.Unlock()
			close(w.tdone)
			return
		}
	}
}

// Stop stops the listener that updates to terminal
func (w *LiveTableWriter) Stop() {
	_ = w.Flush()
	w.tdone <- true
	<-w.tdone
}

func (w *LiveTableWriter) clearRows() {
	_, _ = fmt.Fprint(w.Out, strings.Repeat(clear, w.rowCount))
}

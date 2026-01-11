package util

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

const ESC = 27

// clear the line and move the cursor up
var clear = fmt.Sprintf("%c[%dA%c[2K", ESC, 1, ESC)

// LiveTableWriter is a buffered table writer that updates the terminal.
// The contents of LiveTableWriter will be flushed on a timed interval or
// when Flush is called.
type LiveTableWriter struct {
	// Out is the writer to write the table to.
	Out io.Writer

	buf      bytes.Buffer
	mu       *sync.Mutex
	rowCount int
}

// NewLiveTableWriter returns a new LiveTableWriter with defaults
func NewLiveTableWriter() *LiveTableWriter {
	return &LiveTableWriter{
		Out: io.Writer(os.Stdout),
		mu:  &sync.Mutex{},
	}
}

// Flush writes out the table and resets the buffer. It should be called after
// the last call to Write to ensure that any data buffered in the Writer is
// written to the output. An error is returned if the contents of the buffer
// cannot be written to the underlying output stream.
func (w *LiveTableWriter) Flush() error {
	if len(w.buf.Bytes()) == 0 {
		return nil
	}

	defer w.buf.Reset()
	w.clearRows()

	rows := 0
	for _, b := range w.buf.Bytes() {
		if b == '\n' {
			rows++
		}
	}
	w.rowCount = rows
	_, err := w.Out.Write(w.buf.Bytes())
	if err == nil {
		// Add a new line at the end
		_, err = w.Out.Write([]byte("\n"))
		w.rowCount++
	}
	return err
}

// Write saves the table contents of buf to the writer b.
// The only errors are ones encountered while writing to the underlying buffer.
func (w *LiveTableWriter) Write(buf []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	_ = w.Flush()
	return w.buf.Write(buf)
}

func (w *LiveTableWriter) clearRows() {
	_, _= fmt.Fprint(w.Out, strings.Repeat(clear, w.rowCount))
}

package stream

import (
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats.go"
	"io"
	"strings"
	"sync"
)

type Writer struct {
	conn      *nats.Conn
	subject   string
	closeOnce sync.Once
}

// NewWriter creates a new Writer.
func NewWriter(conn *nats.Conn, subject string) *Writer {
	return &Writer{
		conn:    conn,
		subject: subject,
	}
}

// WriteData writes data to the stream.
func (w *Writer) Write(data []byte) (int, error) {
	msg := &StreamingMsg{
		Type: streamingMsgTypeData,
		Data: data,
	}

	msgData, err := json.Marshal(msg)
	if err != nil {
		return 0, fmt.Errorf("error encoding streaming data: %s", err)
	}

	return len(msgData), w.conn.Publish(w.subject, msgData)
}

// WriteObject writes an object to the stream.
func (w *Writer) WriteObject(obj interface{}) (int, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return 0, fmt.Errorf("error encoding object: %s", err)
	}
	return w.Write(data)
}

// CloseWithCode closes the stream with a specific code.
func (w *Writer) CloseWithCode(code int, text ...string) error {
	return w.doClose(code, strings.Join(text, ": "))
}

// Close closes the stream.
func (w *Writer) Close() error {
	return w.doClose(CloseNormalClosure, "")
}

// doClose closes the stream.
func (w *Writer) doClose(code int, text string) error {
	var err error
	w.closeOnce.Do(func() {
		msg := &StreamingMsg{
			Type: streamingMsgTypeClose,
			CloseError: &CloseError{
				Code: code,
				Text: text,
			},
		}

		var data []byte
		data, err = json.Marshal(msg)
		if err != nil {
			err = fmt.Errorf("error encoding close error: %s", err)
			return
		}

		err = w.conn.Publish(w.subject, data)
	})
	return err
}

// compile time check for interface
var _ = io.WriteCloser(new(Writer))

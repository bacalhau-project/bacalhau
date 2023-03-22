package wasmlogs

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/rs/zerolog/log"
)

const (
	defaultMaxMessages     = 64
	defaultQueueSize       = 16
	defaultWatchQueueSize  = 16
	defaultWatchersAllowed = 4
)

var errClosed = fmt.Errorf("message queue was closed")

// LogMediator implementations are expected to handle the
// writing and persistence of log messages using this interface.
type LogMediator interface {
	// WriteLog is expected to store the provided message and return as
	// quickly as possible, without waiting for confirmation that the
	// message has been flushed to a backing store.
	WriteLog(msg *Message) error

	// ReadLogs will create a channel and return it to the caller, before
	// writing the entire, current, contents of the log file down the channel.
	// The channel will be closed by the mediator once the process has
	// finished, and so the caller must take care to use the 2-var form of
	// reading from a channel.
	ReadLogs() chan *Message

	// WatchLogs will return a channel, and begine writing any newly received
	// log entries, as they are received and before they are written to disk.
	// The channel will be closed by the mediator once the process has finished,
	// and so the caller must take care to use the 2-var form of reading from
	// a channel.
	WatchLogs() chan *Message
}

type LogFileMediator struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	writer     *os.File
	logfile    string
	mq         *MessageQueue
	wg         sync.WaitGroup
	watchers   []*MessageQueue
	closed     bool
}

func NewLogFileMediator(c context.Context, filename string) (*LogFileMediator, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0755) //nolint:gomnd
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(c)

	m := &LogFileMediator{
		ctx:        ctx,
		cancelFunc: cancel,
		writer:     file,
		logfile:    filename,
		watchers:   make([]*MessageQueue, 0, defaultWatchersAllowed),
		mq:         NewMessageQueue(ctx, defaultMaxMessages),
	}

	m.wg.Add(1)
	go m.mediate()
	return m, nil
}

// run will continuously dequeue messages from the internal queue
// and process them by writing to the underlying file.
func (l *LogFileMediator) mediate() {
	defer l.wg.Done()

	var compactBuffer bytes.Buffer

	for !l.closed {
		select {
		case <-l.ctx.Done():
			return
		default:
			msg, err := l.ReadFromQueue()
			if err != nil {
				log.Ctx(l.ctx).Err(err).Msg("error reading from queue")
				break
			}

			if msg == nil {
				log.Ctx(l.ctx).Debug().Msg("reading from queue returned no message, closing")
				l.closed = true
				return
			}

			bytes, err := json.Marshal(msg)
			if err != nil {
				log.Ctx(l.ctx).Err(err).Msg("failed to marshal message into bytes")
				continue
			}

			err = json.Compact(&compactBuffer, bytes)
			if err != nil {
				log.Ctx(l.ctx).Err(err).Msg("failed to compact json into a jsonl")
				continue
			}

			n, err := l.writer.Write(compactBuffer.Bytes())
			if err != nil {
				log.Ctx(l.ctx).Err(err).Msg("failed to write compact representation of message")
				continue
			}
			if n == 0 {
				log.Ctx(l.ctx).Debug().Msg("wrote 0 bytes to message repr")
			}

			// Notify all watchers by enqueueing this message so that they can
			// dequeue it and process it.  If they would block (because they are
			// not moving fast enough) then we will just not notify them of that
			// specific message.
			for _, q := range l.watchers {
				if !q.Full() {
					q.Enqueue(msg)
				}
			}
		}
	}
}

// ReadLogs will return a channel for the caller to read messages from. Then it
// will read all of the log lines from the underlying log file, parse them into
// Message structures and write them down the channel before closing it and then
// stopping it's own goroutine. The caller use the `obj, more <- chan` to be
// sure it isn't waiting around forever for the closed channel to send more.
func (l *LogFileMediator) ReadLogs() chan *Message {
	ch := make(chan *Message, defaultQueueSize)

	go func() {
		defer close(ch)

		f, err := os.OpenFile(l.logfile, os.O_RDONLY, 0755) //nolint:gomnd
		if err != nil {
			log.Ctx(l.ctx).Err(err).Msg("failed to open the log file")
			return
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)

		for {
			select {
			case <-l.ctx.Done():
				return
			default:
				cont := scanner.Scan()
				if !cont {
					return
				}

				msg := new(Message)
				err := json.Unmarshal(scanner.Bytes(), msg)
				if err != nil {
					log.Ctx(l.ctx).Err(err).Msg("failed to parse JSON")
					return
				}

				ch <- msg
			}
		}
	}()

	return ch
}

func (l *LogFileMediator) WatchLogs() *MessageQueue {
	q := NewMessageQueue(l.ctx, defaultWatchQueueSize)
	l.watchers = append(l.watchers, q)
	return q
}

// WriteLog takes the provided message and enqueues it ready
// for reading by a process that will write the message into
// a JSON log file.
func (l *LogFileMediator) WriteLog(msg *Message) error {
	if l.closed {
		return errClosed
	}

	l.mq.Enqueue(msg)
	return nil
}

// Dequeue will continuously remove Messages from the queue
// and write them to a log file.
func (l *LogFileMediator) ReadFromQueue() (*Message, error) {
	if l.closed {
		return nil, errClosed
	}

	msg := l.mq.Dequeue()
	return msg, nil
}

func (l *LogFileMediator) Close() []*Message {
	l.cancelFunc()
	l.closed = true
	l.wg.Wait()

	return l.mq.Drain()
}

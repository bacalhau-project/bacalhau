package wasmlogs

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"

	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
	"github.com/rs/zerolog/log"
)

const (
	DefaultFilePerms = 0755
)

type LogMessageTransformer func(*LogMessage) []byte

type LogReader struct {
	ctx                   context.Context
	file                  *os.File
	reader                *bufio.Reader
	broadcaster           *generic.Broadcaster[*LogMessage]
	rawMessageTransformer LogMessageTransformer
	subscriptionCh        chan *LogMessage
	subscribed            bool
	endOfFileReached      bool
	follow                bool
	streamName            LogStreamType
}

type LogReaderOptions struct {
	ctx                   context.Context
	filename              string
	follow                bool
	streamName            LogStreamType
	rawMessageTransformer LogMessageTransformer
	broadcaster           *generic.Broadcaster[*LogMessage]
}

func NewLogReader(options LogReaderOptions) *LogReader {
	file, err := os.OpenFile(options.filename, os.O_RDONLY, DefaultFilePerms)
	if err != nil {
		log.Ctx(options.ctx).Err(err).Msg("unable to open logfile")
		return nil
	}

	reader := bufio.NewReader(file)
	return &LogReader{
		ctx:                   options.ctx,
		file:                  file,
		reader:                reader,
		broadcaster:           options.broadcaster,
		subscribed:            false,
		rawMessageTransformer: options.rawMessageTransformer,
		streamName:            options.streamName,
		follow:                options.follow,
	}
}

func (r *LogReader) Read(b []byte) (int, error) {
	if r.ctx.Err() != nil {
		return 0, io.EOF
	}

	if !r.endOfFileReached {
		// If we are not at the end of the file, read some more. If we didn't
		// reach the end of the file, then return the (n, error) pair for the
		// successful read.
		n, err := r.readFile(b)
		if !r.endOfFileReached {
			return n, err
		}
	}

	// If at the end of the file, check if we are currently subscribed to the
	// broadcast and subscribe if not. Read the messages and return them as
	// if we read them from the file.
	if r.endOfFileReached && r.follow {
		if !r.subscribed {
			ch, err := r.broadcaster.Subscribe()
			if err != nil {
				return 0, io.EOF
			}
			r.subscriptionCh = ch
			r.subscribed = true
		}

		select {
		case m, more := <-r.subscriptionCh:
			copied := 0
			if more {
				if r.rawMessageTransformer != nil {
					data := r.rawMessageTransformer(m)
					copied = copy(b, data)
				} else {
					copied = copy(b, m.Data)
				}
				return copied, nil
			} else {
				// If the channel was closed, then there is nothing else
				// for us to do.
				return 0, io.EOF
			}
		case <-r.ctx.Done():
			log.Ctx(r.ctx).Debug().Msg("wasm logreaders canceled")
			return 0, io.EOF
		}
	}

	return 0, io.EOF
}

// readFile ..
func (r *LogReader) readFile(b []byte) (int, error) {
	var msg LogMessage

	for {
		if r.ctx.Err() != nil {
			return 0, io.ErrClosedPipe
		}

		data, err := r.reader.ReadBytes('\n')
		if err == io.EOF {
			r.endOfFileReached = true
			return 0, err
		}
		if err != nil {
			return 0, err
		}

		err = json.Unmarshal(data, &msg)
		if err != nil {
			return 0, err
		}

		// It's possible the caller wants a different transforms of the
		// LogMessage than just returning the message data. If we were
		// given one of those functions, use it to transform the data
		// into a format we want.
		if r.rawMessageTransformer != nil {
			data := r.rawMessageTransformer(&msg)
			copied := copy(b, data)
			return copied, nil
		}

		if msg.Stream == r.streamName {
			copied := copy(b, msg.Data)
			return copied, nil
		}
	}

	// unreachable
}

func (r *LogReader) Close() error {
	r.file.Close()
	if r.subscribed {
		r.broadcaster.Unsubscribe(r.subscriptionCh)
	}
	return nil
}

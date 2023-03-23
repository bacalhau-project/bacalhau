package wasmlogs

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
)

type LogReader struct {
	ctx              context.Context
	input            *os.File
	reader           *bufio.Reader
	streamName       string
	endOfFileReached bool
	wantLiveStream   chan bool
	liveStream       chan Message
	follow           bool
}

type LogReaderOptions struct {
	ctx            context.Context
	filename       string
	streamName     string
	liveStream     chan Message
	wantLiveStream chan bool
	follow         bool
}

func NewLogReader(options LogReaderOptions) (*LogReader, error) {
	file, err := os.OpenFile(options.filename, os.O_RDONLY, DefaultFilePerms)
	if err != nil {
		return nil, err
	}

	reader := bufio.NewReader(file)

	return &LogReader{
		ctx:              options.ctx,
		streamName:       options.streamName,
		input:            file,
		reader:           reader,
		endOfFileReached: false,
		liveStream:       options.liveStream,
		wantLiveStream:   options.wantLiveStream,
		follow:           options.follow,
	}, nil
}

func (r *LogReader) Read(b []byte) (int, error) {
	if r.ctx.Err() != nil {
		return 0, io.EOF
	}

	if !r.endOfFileReached {
		// If we are not at the end of the file, read more and we are
		// still not at the end of the file, then return what we did.
		// If we _are_ now at the end of the file, then we will
		// skip this so we can read from the livestream.
		n, err := r.readFile(b)
		if !r.endOfFileReached {
			return n, err
		}
	}

	if r.endOfFileReached && r.follow {
		select {
		case m, more := <-r.liveStream:
			if more {
				copied := copy(b, m.Data)
				return copied, nil
			}
		case <-r.ctx.Done():
			return 0, io.EOF
		}
	}

	return 0, io.EOF
}

func (r *LogReader) readFile(b []byte) (int, error) {
	var msg Message

	// We need to keep reading until we find a record what has a matching
	// stream before we can return it, or we are canceled
	for {
		if r.ctx.Err() != nil {
			return 0, io.ErrClosedPipe
		}

		data, err := r.reader.ReadBytes('\n')
		if err == io.EOF {
			r.endOfFileReached = true
			r.wantLiveStream <- true
			return 0, err
		}
		if err != nil {
			return 0, err
		}

		err = json.Unmarshal(data, &msg)
		if err != nil {
			return 0, err
		}

		if msg.Stream == r.streamName {
			copied := copy(b, msg.Data)
			return copied, nil
		}
	}

	// unreachable
}

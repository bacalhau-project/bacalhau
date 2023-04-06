package wasmlogs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
	"github.com/rs/zerolog/log"
)

type LogManager struct {
	ctx           context.Context
	wg            sync.WaitGroup
	buffer        *generic.RingBuffer[*LogMessage]
	broadcaster   *generic.Broadcaster[*LogMessage]
	file          *os.File
	filename      string
	keepReading   bool
	lifetimeBytes int64
}

func NewLogManager(ctx context.Context, filenameUniquer string) (*LogManager, error) {
	mgr := &LogManager{
		ctx:         ctx,
		buffer:      generic.NewRingBuffer[*LogMessage](0),
		broadcaster: generic.NewBroadcaster[*LogMessage](0), // Use default size
		keepReading: true,
	}
	mgr.wg.Add(1)
	go mgr.logWriter()

	tmpFile, err := os.CreateTemp("", fmt.Sprintf("%s_log.json", filenameUniquer))
	if err != nil {
		return nil, err
	}
	mgr.file = tmpFile
	mgr.filename = tmpFile.Name()

	log.Ctx(ctx).Debug().Msgf("logmanager created logfile: %s", mgr.filename)

	return mgr, nil
}

// logWriter will, until told to stop, dequeue messages from the ring
// buffer and write them to the logfile and broadcast to any subscribing
// channels
func (lm *LogManager) logWriter() {
	defer lm.wg.Done()

	var compactBuffer bytes.Buffer

	for lm.keepReading {
		select {
		case <-lm.ctx.Done():
			lm.keepReading = false
		default:
			compactBuffer.Reset()

			msg := lm.buffer.Dequeue()
			if !lm.keepReading {
				continue
			}

			lm.keepReading = lm.processItem(msg, compactBuffer)
		}
	}

	// We need to drain the remaining items and flush them to the broadcaster
	// and the file
	extra := lm.buffer.Drain()
	log.Ctx(lm.ctx).Debug().Msgf("draining wasm log buffer of %d items", len(extra))
	for _, m := range extra {
		lm.processItem(m, compactBuffer)
	}
}

func (lm *LogManager) processItem(msg *LogMessage, compactBuffer bytes.Buffer) bool {
	if msg == nil {
		// We have a sentinel on close so make sure we don't try and
		// process it.
		log.Ctx(lm.ctx).Debug().Msg("logmanager received sentinel exit message")
		return false
	}

	lm.lifetimeBytes += int64(len(msg.Data))

	// Broadcast the message to anybody that might be listening
	_ = lm.broadcaster.Broadcast(msg)

	// Convert the message to JSON and write it to the log file
	data, err := json.Marshal(msg)
	if err != nil {
		log.Ctx(lm.ctx).Err(err).Msg("failed to unmarshall a wasm log message")
		return true
	}

	err = json.Compact(&compactBuffer, data)
	if err != nil {
		log.Ctx(lm.ctx).Err(err).Msg("failed to compact wasm log message")
		return true
	}
	compactBuffer.Write([]byte{'\n'})

	// write msg to file and also broadcast the message
	wrote, err := lm.file.Write(compactBuffer.Bytes())
	if err != nil {
		log.Ctx(lm.ctx).Err(err).Msgf("failed to write wasm log to file: %s", lm.file.Name())
		return true
	}
	if wrote == 0 {
		log.Ctx(lm.ctx).Debug().Msgf("zero byte write in wasm logging to: %s", lm.file.Name())
		return true
	}

	return true
}

func (lm *LogManager) GetWriters() (io.Writer, io.Writer) {
	writerFunc := func(strm LogStreamType) func([]byte) *LogMessage {
		return func(b []byte) *LogMessage {
			m := LogMessage{
				Timestamp: time.Now().Unix(),
				Stream:    strm,
			}
			m.Data = append([]byte(nil), b...)
			return &m
		}
	}

	stdout := NewLogWriter(lm.buffer, writerFunc(LogStreamStdout))
	stderr := NewLogWriter(lm.buffer, writerFunc(LogStreamStderr))
	return stdout, stderr
}

func (lm *LogManager) GetDefaultReaders(follow bool) (io.Reader, io.Reader) {
	stdout := NewLogReader(LogReaderOptions{
		ctx:                   lm.ctx,
		filename:              lm.file.Name(),
		follow:                follow,
		rawMessageTransformer: nil,
		broadcaster:           lm.broadcaster,
		streamName:            LogStreamStdout,
	})

	stderr := NewLogReader(LogReaderOptions{
		ctx:                   lm.ctx,
		filename:              lm.file.Name(),
		follow:                follow,
		rawMessageTransformer: nil,
		broadcaster:           lm.broadcaster,
		streamName:            LogStreamStderr,
	})

	return stdout, stderr
}

func (lm *LogManager) GetMuxedReader(follow bool) io.ReadCloser {
	transformer := func(msg *LogMessage) []byte {
		tag := logger.StdoutStreamTag
		if msg.Stream == LogStreamStderr {
			tag = logger.StderrStreamTag
		}
		df := logger.NewDataFrameFromData(tag, msg.Data)
		return df.ToBytes()
	}

	return NewLogReader(LogReaderOptions{
		ctx:                   lm.ctx,
		filename:              lm.file.Name(),
		follow:                follow,
		rawMessageTransformer: transformer,
		broadcaster:           lm.broadcaster,
		streamName:            LogStreamStdout,
	})
}

func (lm *LogManager) Close() {
	lm.keepReading = false
	lm.buffer.Enqueue(nil)

	// Wait for completion before we remove the log file
	lm.wg.Wait()

	log.Ctx(lm.ctx).Debug().Msgf("logmanager removing logfile: %s", lm.filename)
	os.Remove(lm.filename)
}

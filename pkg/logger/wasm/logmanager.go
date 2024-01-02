package wasmlogs

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/util"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
	"github.com/rs/zerolog/log"
)

const (
	DefaultBufferChannelLen = 4096
)

type LogManager struct {
	ctx           context.Context
	wg            sync.WaitGroup
	buffer        *generic.RingBuffer[*LogMessage]
	broadcaster   *generic.Broadcaster[*LogMessage]
	file          *os.File
	filename      string
	executionID   string
	keepReading   bool
	lifetimeBytes int64
}

func NewLogManager(ctx context.Context, executionID string) (*LogManager, error) {
	mgr := &LogManager{
		ctx:         ctx,
		buffer:      generic.NewRingBuffer[*LogMessage](0),
		broadcaster: generic.NewBroadcaster[*LogMessage](0), // Use default size
		keepReading: true,
		executionID: executionID,
	}
	mgr.wg.Add(1)
	go mgr.logWriter()

	tmpFile, err := os.CreateTemp("", fmt.Sprintf("%s_log.json", executionID))
	if err != nil {
		return nil, err
	}
	mgr.file = tmpFile
	mgr.filename = tmpFile.Name()

	log.Ctx(ctx).Debug().Str("Execution", executionID).Msgf("logmanager created logfile: %s", mgr.filename)

	return mgr, nil
}

// logWriter will, until told to stop, dequeue messages from the ring
// buffer and write them to the logfile and broadcast to any subscribing
// channels
func (lm *LogManager) logWriter() {
	defer lm.wg.Done()

	messageCount := 0

	// Use a goroutine to dequeue from the ringbuffer, and write to the channel
	// we gave it
	messageChan := make(chan *LogMessage, DefaultBufferChannelLen)
	lm.wg.Add(1)
	go lm.dequeueToChannel(messageChan)

	for lm.keepReading {
		select {
		case <-lm.ctx.Done():
			lm.keepReading = false
		case msg, more := <-messageChan:
			if !more {
				lm.keepReading = false
				continue
			}

			lm.keepReading = lm.processItem(msg)
			messageCount += 1
		}
	}
}

// Reads from the ringbuffer as quickly as it can and then writes
// the messages down the buffered channel as quickly as possible.
func (lm *LogManager) dequeueToChannel(ch chan *LogMessage) {
	defer lm.wg.Done()

	for lm.keepReading {
		msg := lm.buffer.Dequeue()
		if msg == nil {
			break
		}
		ch <- msg
	}

	close(ch)
}

func (lm *LogManager) processItem(msg *LogMessage) bool {
	if msg == nil {
		// We have a sentinel on close so make sure we don't try and
		// process it.
		log.Ctx(lm.ctx).Debug().Str("Execution", lm.executionID).Msg("logmanager received sentinel exit message")
		return false
	}

	lm.lifetimeBytes += int64(len(msg.Data))

	// Broadcast the message to anybody that might be listening
	_ = lm.broadcaster.Broadcast(msg)

	// Don't bother encoding to json when we know what the json line looks like, we
	// may as well just sprintf it into place and base64 encoding the bytes
	str := fmt.Sprintf("{\"s\":%d,\"d\":\"%s\",\"t\":%d}\n",
		msg.Stream,
		base64.StdEncoding.EncodeToString(msg.Data),
		msg.Timestamp)

	// write msg to file and also broadcast the message
	wrote, err := lm.file.Write([]byte(str))
	if err != nil {
		log.Ctx(lm.ctx).Err(err).Str("Execution", lm.executionID).Msgf("failed to write wasm log to file: %s", lm.file.Name())
		return true
	}
	if wrote == 0 {
		log.Ctx(lm.ctx).Debug().Str("Execution", lm.executionID).Msgf("zero byte write in wasm logging to: %s", lm.file.Name())
		return true
	}

	return true
}

func (lm *LogManager) Drain() {
	lm.keepReading = false
	lm.buffer.Enqueue(nil)

	// We need to drain the remaining items and flush them to the broadcaster
	// and the file
	extra := lm.buffer.Drain()
	log.Ctx(lm.ctx).Debug().Str("Execution", lm.executionID).Msgf("draining wasm log buffer of %d items", len(extra))
	for _, m := range extra {
		lm.processItem(m)
	}

	// Ask the file to sync to disk
	_ = lm.file.Sync()

	info, err := lm.file.Stat()
	if err == nil {
		log.Ctx(lm.ctx).
			Debug().
			Str("Size", strconv.Itoa(int(info.Size()))).
			Str("Execution", lm.executionID).
			Msg("finished writing to logfile")
	}
}

func (lm *LogManager) GetWriters() (io.WriteCloser, io.WriteCloser) {
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
	// Wait for completion before we remove the log file
	lm.keepReading = false
	lm.buffer.Enqueue(nil)
	lm.wg.Wait()

	go func(ctx context.Context, executionID string, filename string) {
		tensecs := time.After(time.Duration(10) * time.Second) //nolint:gomnd
		<-tensecs

		log.Ctx(ctx).Debug().Msgf("logmanager removing logfile for %s: %s", executionID, filename)
		os.Remove(lm.filename)
	}(util.NewDetachedContext(lm.ctx), lm.executionID, lm.filename)
}

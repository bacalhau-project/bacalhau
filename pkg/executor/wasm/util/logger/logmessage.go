package wasmlogs

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"math"
)

type LogStreamType int8

const (
	LogStreamStdout LogStreamType = 1
	LogStreamStderr LogStreamType = 2
)

const (
	LogMessageHeaderLength    = 17
	LogMessageStreamOffset    = 4
	LogMessageTimestampOffset = 5
	LogMessageDataSizeOffset  = 13
)

type LogMessage struct {
	Stream    LogStreamType `json:"s"`
	Data      []byte        `json:"d"`
	Timestamp int64         `json:"t"`
}

// ToBytes will convert the current LogMessage into a byte array which can
// be written to disk and reconstituted by FromBytes.
//
// The first 4 bytes are the size of the entire LogMessage
// The next byte is O for stdout, E for stderr.
// The next 8bytes are the timestamp
// The next 4 bytes (n) are the size of the data
// Then the remaining n bytes are the data
// TODO: Ignoring lint for now since fixing it is a task by itself.
//
//nolint:gosec // G115: intentionally converting int to uint32 for message size
func (m *LogMessage) ToBytes() []byte {
	size := uint32(LogMessageHeaderLength + len(m.Data))
	b := make([]byte, size)

	binary.BigEndian.PutUint32(b, size)

	b[LogMessageStreamOffset] = byte(m.Stream)

	binary.BigEndian.PutUint64(b[LogMessageTimestampOffset:], uint64(m.Timestamp))
	binary.BigEndian.PutUint32(b[LogMessageDataSizeOffset:], uint32(len(m.Data)))

	_ = copy(b[LogMessageHeaderLength:], m.Data)
	return b
}

func (m *LogMessage) FromReader(reader bufio.Reader) error {
	sizeB, err := reader.Peek(4) //nolint:mnd
	if err != nil {
		return err
	}

	size := binary.BigEndian.Uint32(sizeB)
	buffer := make([]byte, size)

	read, err := reader.Read(buffer)
	if err != nil {
		return err
	}
	if read != len(buffer) {
		return fmt.Errorf("short read of logmessage from reader: expected %d got %d", len(buffer), read)
	}

	m.Stream = LogStreamType(buffer[LogMessageStreamOffset])

	timestamp := binary.BigEndian.Uint64(buffer[LogMessageTimestampOffset:])
	if timestamp > math.MaxInt64 {
		return fmt.Errorf("timestamp value %d exceeds maximum int64 value", timestamp)
	}
	m.Timestamp = int64(timestamp)

	m.Data = append([]byte(nil), buffer[LogMessageHeaderLength:]...)

	return nil
}

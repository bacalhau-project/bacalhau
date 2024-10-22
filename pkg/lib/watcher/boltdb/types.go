package boltdb

import (
	"encoding/binary"
	"errors"
)

const (
	seqNumBytes    = 8
	timestampBytes = 8
)

type eventKey struct {
	SeqNum    uint64
	Timestamp int64
}

// newEventKey creates a new event key.
func newEventKey(seqNum uint64, timestamp int64) *eventKey {
	return &eventKey{
		SeqNum:    seqNum,
		Timestamp: timestamp,
	}
}

//nolint:gosec // G115: limits within reasonable bounds
func (k *eventKey) MarshalBinary() ([]byte, error) {
	buf := make([]byte, seqNumBytes+timestampBytes)
	binary.BigEndian.PutUint64(buf[:seqNumBytes], k.SeqNum)
	binary.BigEndian.PutUint64(buf[seqNumBytes:], uint64(k.Timestamp))
	return buf, nil
}

//nolint:gosec // G115: limits within reasonable bounds
func (k *eventKey) UnmarshalBinary(data []byte) error {
	if len(data) != seqNumBytes+timestampBytes {
		return errors.New("invalid event key length")
	}
	k.SeqNum = binary.BigEndian.Uint64(data[:seqNumBytes])
	k.Timestamp = int64(binary.BigEndian.Uint64(data[seqNumBytes:]))
	return nil
}

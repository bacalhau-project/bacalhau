//go:build unit || !integration

package boltdb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEventKey(t *testing.T) {
	t.Run("MarshalBinary and UnmarshalBinary", func(t *testing.T) {
		// Create a new eventKey
		now := time.Now().UnixNano()
		originalKey := newEventKey(12345, now)

		// Marshal the key
		data, err := originalKey.MarshalBinary()
		assert.NoError(t, err)
		assert.Len(t, data, 16) // 8 bytes for SeqNum + 8 bytes for Timestamp

		// Unmarshal the key
		var unmarshaledKey eventKey
		err = unmarshaledKey.UnmarshalBinary(data)
		assert.NoError(t, err)

		// Check if the unmarshaled key matches the original
		assert.Equal(t, originalKey.SeqNum, unmarshaledKey.SeqNum)
		assert.Equal(t, originalKey.Timestamp, unmarshaledKey.Timestamp)
	})

	t.Run("UnmarshalBinary with invalid data length", func(t *testing.T) {
		var key eventKey
		err := key.UnmarshalBinary([]byte{1, 2, 3}) // Less than 16 bytes
		assert.Error(t, err)
		assert.Equal(t, "invalid event key length", err.Error())
	})

	t.Run("newEventKey", func(t *testing.T) {
		seqNum := uint64(54321)
		timestamp := time.Now().UnixNano()
		key := newEventKey(seqNum, timestamp)

		assert.Equal(t, seqNum, key.SeqNum)
		assert.Equal(t, timestamp, key.Timestamp)
	})
}

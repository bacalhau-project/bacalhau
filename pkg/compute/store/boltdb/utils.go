package boltdb

import (
	"encoding/binary"

	bolt "go.etcd.io/bbolt"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const (
	// uint64ByteSize is the number of bytes needed to represent a uint64
	uint64ByteSize = 8
)

// strToBytes converts a string to a byte slice
func strToBytes(s string) []byte {
	return []byte(s)
}

// uint64ToBytes converts an uint64 to a byte slice
func uint64ToBytes(i uint64) []byte {
	buf := make([]byte, uint64ByteSize)
	binary.BigEndian.PutUint64(buf, i)
	return buf
}

// bytesToUint64 converts a byte slice to an uint64
func bytesToUint64(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

// Helper function to safely access buckets and nested buckets
func bucket(tx *bolt.Tx, bucketNames ...string) *bolt.Bucket {
	var b *bolt.Bucket
	for _, name := range bucketNames {
		if b == nil {
			b = tx.Bucket([]byte(name))
		} else {
			b = b.Bucket([]byte(name))
		}
		if b == nil {
			return nil
		}
	}
	return b
}

// isEmptyBucket checks if a bucket is empty
func isEmptyBucket(b *bolt.Bucket) bool {
	k, _ := b.Cursor().First()
	return k == nil
}

func stateBucketKey(execution *models.Execution) []byte {
	return strToBytes(stateBucketKeyStr(execution))
}

func stateBucketKeyStr(execution *models.Execution) string {
	return execution.ComputeState.StateType.String()
}

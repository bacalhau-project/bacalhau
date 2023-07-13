package boltjobstore

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)

const (
	DefaultDatabasePermissions   = 0600
	DefaultBucketSearchSliceSize = 16
)

func GetDatabase(path string) (*bolt.DB, error) {
	database, err := bolt.Open(path, DefaultDatabasePermissions, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return nil, err
	}
	return database, nil
}

// GetBucketsWithPartialName will search through the provided bucket to find other buckets with
// a name that starts with the partialname that is provided.
func GetBucketsWithPartialName(tx *bolt.Tx, bucket *bolt.Bucket, partialName []byte) ([][]byte, error) {
	bucketNames := make([][]byte, 0, DefaultBucketSearchSliceSize)

	err := bucket.ForEachBucket(func(k []byte) error {
		if bytes.HasPrefix(k, partialName) {
			bucketNames = append(bucketNames, k)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return bucketNames, nil
}

// GetBucketData is a helper that will use the provided details to find
// a key in a specific bucket and return its data.
func GetBucketData(tx *bolt.Tx, bucketPath string, key []byte) []byte {
	b, err := NewBucketPath(bucketPath).Get(tx, false)
	if err != nil {
		return nil
	}
	return b.Get(key)
}

type BucketPath struct {
	path string
}

// NewBucketPath creates a bucket path which can be used to describe the
// nested relationship between buckets, rather than calling b.Bucket() on
// each b found.  BucketPaths are typically described using strings like
// "root.bucket.here".
func NewBucketPath(sections ...string) *BucketPath {
	return &BucketPath{
		path: strings.Join(sections, "."),
	}
}

// Get retrieves the Bucket, or an error, for the bucket found at this path
func (bp *BucketPath) Get(tx *bolt.Tx, create bool) (*bolt.Bucket, error) {
	var err error
	path := strings.Split(bp.path, ".")

	type BucketMaker interface {
		Bucket([]byte) *bolt.Bucket
		CreateBucketIfNotExists([]byte) (*bolt.Bucket, error)
	}

	getBucket := func(root BucketMaker, name string) (*bolt.Bucket, error) {
		bucket := root.Bucket([]byte(name))
		return bucket, nil
	}
	if create {
		getBucket = func(root BucketMaker, name string) (*bolt.Bucket, error) {
			bucket, err := root.CreateBucketIfNotExists([]byte(name))
			if err != nil {
				return nil, err
			}
			return bucket, nil
		}
	}

	bucket, err := getBucket(tx, path[0])
	if err != nil {
		return nil, err
	}
	if bucket == nil {
		return nil, bolt.ErrBucketNotFound
	}

	for _, name := range path[1:] {
		sub, err := getBucket(bucket, name)
		if err != nil {
			return nil, err
		}
		if sub == nil {
			return nil, bolt.ErrBucketNotFound
		}
		bucket = sub
	}

	return bucket, nil
}

// BucketSequenceString returns the next sequence in the provided
// bucket, formatted as a 3 character padded string to ensure that
// bolt's lexicographic ordering will return them in the correct
// order
func BucketSequenceString(_ *bolt.Tx, bucket *bolt.Bucket) string {
	seqNum, err := bucket.NextSequence()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%03d", seqNum)
}

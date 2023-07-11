package boltjobstore

import (
	"fmt"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)

const (
	DefaultDatabasePermissions = 0600
)

func GetDatabase(path string) (*bolt.DB, error) {
	database, err := bolt.Open(path, DefaultDatabasePermissions, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return nil, err
	}
	return database, nil
}

func GetBucketData(tx *bolt.Tx, bucketPath string, key []byte) []byte {
	b, err := GetBucketByPath(tx, bucketPath, false)
	if err != nil {
		return nil
	}
	return b.Get(key)
}

// GetSubBucket accepts a bucket path which is a . separated string showing a path
// from the first item (the top level bucket) to the last item, the final bucket
// which will be returned. If at any time a bucket isn't found then an error will
// be returned, unless `create` is set to true in which case the missing bucket
// will be created.
func GetBucketByPath(tx *bolt.Tx, bucketPath string, create bool) (*bolt.Bucket, error) {
	var err error
	path := strings.Split(bucketPath, ".")

	type BucketMaker interface {
		Bucket([]byte) *bolt.Bucket
		CreateBucketIfNotExists([]byte) (*bolt.Bucket, error)
	}

	getBucket := func(root BucketMaker, name string) (*bolt.Bucket, error) {
		bucket := root.Bucket([]byte(name))
		if bucket == nil {
			return nil, bolt.ErrBucketNotFound
		}
		return bucket, nil
	}
	if create {
		getBucket = func(root BucketMaker, name string) (*bolt.Bucket, error) {
			bucket, err := root.CreateBucketIfNotExists([]byte(name))
			if err != nil {
				return nil, err
			}
			if bucket == nil {
				return nil, bolt.ErrBucketNotFound
			}
			return bucket, nil
		}
	}

	bucket, err := getBucket(tx, path[0])
	if err != nil {
		return nil, err
	}

	for _, name := range path[1:] {
		sub, err := getBucket(bucket, name)
		if err != nil {
			return nil, err
		}
		bucket = sub
	}

	return bucket, nil
}

// PutBucketSentinel adds a sentinel at bucketPath for the provided key
func PutBucketSentinel(tx *bolt.Tx, bucketPath string, key []byte) error {
	b, err := GetBucketByPath(tx, bucketPath, true)
	if err != nil {
		return err
	}

	return b.Put(key, nil)
}

// GetBucketSentinels, for cases where we are using a bucket to hold keys
// with no values, will return all of the keys within a specific bucket
func GetBucketSentinels(tx *bolt.Tx, bucketPath string) ([][]byte, error) {
	b, err := GetBucketByPath(tx, bucketPath, true)
	if err != nil {
		return nil, err
	}

	var results [][]byte

	err = b.ForEach(func(k []byte, v []byte) error {
		results = append(results, k)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return results, nil
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

package boltjobstore

import (
	"fmt"
	"strings"

	bolt "go.etcd.io/bbolt"
)

type BucketPath struct {
	path string
}

// NewBucketPath creates a bucket path which can be used to describe the
// nested relationship between buckets, rather than calling b.Bucket() on
// each b found.  BucketPaths are typically described using strings like
// "root.bucket.here".
func NewBucketPath(sections ...string) *BucketPath {
	return &BucketPath{
		path: strings.Join(sections, BucketPathDelimiter),
	}
}

// Get retrieves the Bucket, or an error, for the bucket found at this path
func (bp *BucketPath) Get(tx *bolt.Tx, create bool) (*bolt.Bucket, error) {
	path := strings.Split(bp.path, BucketPathDelimiter)

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

	var bucket *bolt.Bucket
	var bucketMaker BucketMaker = tx

	for _, name := range path {
		sub, err := getBucket(bucketMaker, name)
		if err != nil {
			return nil, err
		}
		if sub == nil {
			return nil, bolt.ErrBucketNotFound
		}
		bucket = sub
		bucketMaker = sub
	}

	return bucket, nil
}

func (bp *BucketPath) Sub(names ...[]byte) *BucketPath {
	path := bp.path
	for _, s := range names {
		path = fmt.Sprintf("%s%s%s", path, BucketPathDelimiter, s)
	}
	return NewBucketPath(path)
}

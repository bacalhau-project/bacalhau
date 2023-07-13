package boltjobstore

import (
	bolt "go.etcd.io/bbolt"
)

// Index is a bucket type that encodes both a label and an identifier,
// for use as a sentinel marker to show the presence of a thing. For example
// an index for job `94b136a3` having label `gpu`, we would create the
// `gpu` bucket if it didn't exist, and then a bucket with the job ID.
type Index struct {
	rootBucketPath *BucketPath
}

func NewIndex(bucketPath string) *Index {
	return &Index{
		rootBucketPath: NewBucketPath(bucketPath),
	}
}

func (i *Index) Add(tx *bolt.Tx, label []byte, identifier []byte) error {
	bkt, err := i.rootBucketPath.Get(tx, true)
	if err != nil {
		return err
	}

	bktLabel, err := bkt.CreateBucketIfNotExists(label)
	if err != nil {
		return err
	}

	_, err = bktLabel.CreateBucketIfNotExists(identifier)
	return err
}

func (i *Index) List(tx *bolt.Tx, label []byte) ([][]byte, error) {
	bkt, err := i.rootBucketPath.Get(tx, true)
	if err != nil {
		return nil, err
	}

	lblBkt := bkt.Bucket(label)
	if lblBkt == nil {
		return nil, bolt.ErrBucketNotFound
	}

	result := make([][]byte, 0, DefaultBucketSearchSliceSize)
	err = lblBkt.ForEachBucket(func(k []byte) error {
		result = append(result, k)
		return nil
	})
	return result, err
}

func (i *Index) Remove(tx *bolt.Tx, label []byte, identifier []byte) error {
	bkt, err := i.rootBucketPath.Get(tx, true)
	if err != nil {
		return err
	}

	lblBkt := bkt.Bucket(label)
	if lblBkt == nil {
		return bolt.ErrBucketNotFound
	}

	return lblBkt.DeleteBucket(identifier)
}

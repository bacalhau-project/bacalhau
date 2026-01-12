package boltjobstore

import (
	"errors"

	bolt "go.etcd.io/bbolt"
)

// Index is a bucket type that encodes both a label and an identifier,
// for use as a sentinel marker to show the presence of a thing. For example
// an index for job `94b136a3` having label `gpu`, we would create the
// `gpu` bucket if it didn't exist, and then a bucket with the job ID.
//
// Most methods will take a label, and an identifier and these serve as
// the attenuating field and the item itself.  So for a client id index,
// where we want to have a list of job ids for each client, the index
// will look like
//
//	jobs_clients
//	   |----- CLIENT ID 1
//	             |---- JOBID 1
//	             |---- JOBID 2
//	   |----- CLIENT ID 2
//	    .....
//
// In this case, JOBID1 is the identifier, and CLIENT ID 1 is the subpath/label.
// In some cases, such as indices for jobs in a specific state, we may
// not have/need the label and so subpath can be excluded instead.
// The primary use of this is currently the list of InProgress jobs where
// there is no attenuator (e.g. clientid)
type Index struct {
	rootBucketPath *BucketPath
}

func NewIndex(bucketPath string) *Index {
	return &Index{
		rootBucketPath: NewBucketPath(bucketPath),
	}
}

func (i *Index) Add(tx *bolt.Tx, identifier []byte, subpath ...[]byte) error {
	bkt, err := i.rootBucketPath.Sub(subpath...).Get(tx, true)
	if err != nil {
		return err
	}

	return bkt.Put(identifier, []byte(""))
}

func (i *Index) List(tx *bolt.Tx, subpath ...[]byte) ([][]byte, error) {
	bkt, err := i.rootBucketPath.Sub(subpath...).Get(tx, false)
	if err != nil && !errors.Is(err, bolt.ErrBucketNotFound) { //nolint:staticcheck // TODO: migrate to bbolt/errors package
		return nil, err
	}

	result := make([][]byte, 0, DefaultBucketSearchSliceSize)
	if bkt == nil {
		// If the bucket we are looking to list does not exist, then
		// return the empty results
		return result, nil
	}

	err = bkt.ForEach(func(k []byte, _ []byte) error {
		result = append(result, k)
		return nil
	})
	return result, err
}

func (i *Index) Remove(tx *bolt.Tx, identifier []byte, subpath ...[]byte) error {
	bkt, err := i.rootBucketPath.Sub(subpath...).Get(tx, false)
	if err != nil {
		return err
	}

	return bkt.Delete(identifier)
}

package boltjobstore

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	bolt "go.etcd.io/bbolt"
)

const (
	DefaultDatabasePermissions   = 0600
	DefaultBucketSearchSliceSize = 16
	BucketPathDelimiter          = "/"
	DefaultJobIDListSize         = 32
	DeadJobBucket                = "deadjobs"
)

func GetDatabase(path string) (*bolt.DB, error) {
	database, err := bolt.Open(path, DefaultDatabasePermissions, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return nil, err
	}
	return database, nil
}

// GetBucketsByPrefix will search through the provided bucket to find other buckets with
// a name that starts with the partialname that is provided.
func GetBucketsByPrefix(tx *bolt.Tx, bucket *bolt.Bucket, partialName []byte) ([][]byte, error) {
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
func GetBucketData(tx *bolt.Tx, bucketPath *BucketPath, key []byte) []byte {
	bkt, err := bucketPath.Get(tx, false)
	if err != nil {
		return nil
	}
	return bkt.Get(key)
}

// BucketSequenceString returns the next sequence in the provided
// bucket, formatted as a 16 character padded string to ensure that
// bolt's lexicographic ordering will return them in the correct
// order
func BucketSequenceString(_ *bolt.Tx, bucket *bolt.Bucket) string {
	seqNum, err := bucket.NextSequence()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%016d", seqNum)
}

// MarkDeadJobs identifies dead jobs and returns their job id
//
// Once a job has reached a terminal state it should have a limited
// lifetime, after which it should be deleted. Each jobtype will have
// a different duration, and if not present in the lifetimes map then
// it will never be deleted.
func FindDeadJobs(readTx *bolt.Tx, now time.Time, lifetimes map[string]time.Duration) ([]string, error) {
	jobs, err := NewBucketPath(BucketJobs).Get(readTx, false)
	if err != nil {
		return nil, err
	}

	jobids := make([]string, 0, DefaultJobIDListSize)

	err = jobs.ForEachBucket(func(k []byte) (err error) {
		jobBkt := jobs.Bucket(k)
		if jobBkt == nil {
			return fmt.Errorf("failed to load bucket while iterating")
		}

		var job model.Job
		var state model.JobState

		if specBytes := jobBkt.Get(SpecKey); specBytes == nil {
			return fmt.Errorf("failed to load job while iterating jobs")
		} else {
			err = json.Unmarshal(specBytes, &job)
			if err != nil {
				return err
			}
		}

		lifetime, ok := lifetimes[job.Type()]
		if !ok {
			// No duration available for this job type, so we can safely
			// just move onto the next
			return nil
		}

		if stateBytes := jobBkt.Get(StateKey); stateBytes == nil {
			return fmt.Errorf("failed to load job state while iterating jobs")
		} else {
			err = json.Unmarshal(stateBytes, &state)
			if err != nil {
				return err
			}
		}

		if state.State.IsTerminal() {
			if now.Unix() > state.UpdateTime.Add(lifetime).Unix() {
				jobids = append(jobids, state.JobID)
			}
		}

		return nil
	})

	return jobids, err
}

// DeleteDeadJobs deletes all of the specified jobIDs by deleting the bucket
// with a name matching each jobid within the jobs bucket
func DeleteDeadJobs(writeTX *bolt.Tx, jobIDs []string) error {
	errList := make([]error, 0, len(jobIDs))

	if bkt, err := NewBucketPath(BucketJobs).Get(writeTX, false); err != nil {
		return err
	} else {
		for _, jobID := range jobIDs {
			e := bkt.DeleteBucket([]byte(jobID))
			if e != nil {
				errList = append(errList, e)
			}
		}
	}

	if len(errList) > 0 {
		return fmt.Errorf("failed to delete the following jobs: %s", errors.Join(errList...))
	}

	return nil
}

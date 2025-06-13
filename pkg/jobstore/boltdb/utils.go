package boltjobstore

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// ExecutionJobPair represents a pair of execution ID and job ID
type ExecutionJobPair struct {
	ExecutionID string
	JobID       string
}

// encodeExecutionJobKey creates a composite key from execution ID and job ID
func encodeExecutionJobKey(executionID, jobID string) string {
	return executionID + ":" + jobID
}

// decodeExecutionJobKey extracts execution ID and job ID from a composite key
func decodeExecutionJobKey(compositeKey string) (executionID, jobID string, err error) {
	parts := strings.Split(compositeKey, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid composite key format: %s", compositeKey)
	}
	return parts[0], parts[1], nil
}

// splitInProgressIndexKey returns the job type and the job index from
// the in-progress index key. If no delimiter is found, then this index
// was created before this feature was implemented, and we are unable
// to filter on its type so will return "" as the type.
func splitInProgressIndexKey(key string) (string, string) {
	parts := strings.Split(key, ":")
	if len(parts) == 1 {
		return key, ""
	}

	k, typ := parts[1], parts[0]
	return k, typ
}

// createInProgressIndexKey will create a composite key for the in-progress index
func createInProgressIndexKey(job *models.Job) string {
	return fmt.Sprintf("%s:%s", job.Type, job.ID)
}

// createJobNameIndexKey will create a composite key for the job-names index
func createJobNameIndexKey(name string, namespace string) string {
	return fmt.Sprintf("%s:%s", name, namespace)
}

// uint64ToBytes converts an uint64 to a byte slice
func uint64ToBytes(i uint64) []byte {
	//nolint:mnd
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, i)
	return buf
}

// bytesToUint64 converts a byte slice to an uint64
func bytesToUint64(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

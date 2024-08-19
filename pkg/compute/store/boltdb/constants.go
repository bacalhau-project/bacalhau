package boltdb

const (
	defaultSliceRetrievalCapacity = 10
	defaultDatabasePermissions    = 0600
)

// Schema versioning
// The currentSchemaVersion is used as a prefix for all bucket names to support
// future schema changes while maintaining backward compatibility. When the data
// structure needs to be modified in a way that's not backward compatible, increment
// this version and create new buckets with the updated schema.
//
// This approach allows for:
// 1. Clear separation between different schema versions
// 2. Easier data migrations (old version buckets can coexist with new ones)
// 3. Gradual rollout of schema changes
// 4. Ability to roll back to previous schema version if issues are encountered
//
// To introduce a new schema version:
//  1. Increment the currentSchemaVersion (e.g., to "v2"). This will auto create new buckets.
//  2. Implement a migration strategy to move data from old buckets to new ones
//  3. The store should now read/write data from the new buckets
const currentSchemaVersion = "v1"

// Bucket names
// All bucket names are prefixed with the current schema version to support
// versioning and backward compatibility
const (
	// Main buckets
	executionsBucket      = currentSchemaVersion + "_executions"
	executionEventsBucket = currentSchemaVersion + "_execution_events"

	// Index buckets
	idxExecutionsByJobBucket   = currentSchemaVersion + "_idx_executions_by_job_id"
	idxExecutionsByStateBucket = currentSchemaVersion + "_idx_executions_by_state"

	// Event-related buckets
	eventsBucket      = currentSchemaVersion + "_events"
	checkpointsBucket = currentSchemaVersion + "_checkpoints"
)

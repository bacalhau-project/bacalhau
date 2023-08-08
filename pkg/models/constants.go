package models

const (
	// DefaultNamespace is the default namespace.
	DefaultNamespace = "default"
)

const (
	// JobTypeService represents a long-running job that runs on a desired number of nodes
	// matching the specified constraints.
	JobTypeService = "service"

	// JobTypeDaemon represents a long-running job that runs on all nodes matching the
	// specified constraints.
	JobTypeDaemon = "daemon"

	// JobTypeBatch represents a batch job that runs to completion on the desired number
	// of nodes matching the specified constraints.
	JobTypeBatch = "batch"

	// JobTypeOps represents a batch job that runs to completion on all nodes matching
	// the specified constraints.
	JobTypeOps = "ops"
)

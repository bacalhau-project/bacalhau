package model

import (
	"math"
	"time"
)

const (
	// DefaultNamespace is the default namespace.
	DefaultNamespace = "default"
)

const (
	JobTypeService = "service"
	JobTypeBatch   = "batch"
	JobTypeOps     = "ops" // TODO: revisit the job naming
	JobTypeSystem  = "system"
)

// The user did not specify a timeout or explicitly requested that the job
// should receive the longest possible timeout. This value will be changed
// by the node into whatever is configured as max timeout.
const DefaultJobTimeout time.Duration = time.Duration(0)

// NoJobTimeout specifies that the job should not be subject to timeouts. This
// value is the largest possible time.Duration that is a whole number of seconds
// so conversions into an int64 number of seconds and back again are bijective.
var NoJobTimeout time.Duration = time.Duration(math.MaxInt64).Truncate(time.Second)

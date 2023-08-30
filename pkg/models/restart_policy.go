package models

import (
	"math"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type RestartPolicy struct {
	Attempts int
}

// NewRestartPolicy creates a restart policy based on the provided jobtype
// which can be used by the compute node to retry jobs locally in the
// case of failure.
func NewRestartPolicy(typ string) *RestartPolicy {
	var attempts int

	switch typ {
	case JobTypeService, JobTypeDaemon:
		attempts = math.MaxInt
	default: // JobTypeBatch, JobTypeOps
		attempts = 1
	}

	return &RestartPolicy{
		Attempts: attempts,
	}
}

// NewDefaultRestartPolicy returns a default restart policy, which is one
// attempt at executing, and then fail.
func NewDefaultRestartPolicy() *RestartPolicy {
	return NewRestartPolicy(model.JobTypeBatch)
}

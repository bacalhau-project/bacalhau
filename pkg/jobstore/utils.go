package jobstore

import (
	jobutils "github.com/bacalhau-project/bacalhau/pkg/job"
)

func GetStateResolver(db Store) *jobutils.StateResolver {
	return jobutils.NewStateResolver(
		db.GetJob,
		db.GetJobState,
	)
}

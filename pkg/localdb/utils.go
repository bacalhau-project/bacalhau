package localdb

import (
	jobutils "github.com/filecoin-project/bacalhau/pkg/job"
)

func GetStateResolver(db LocalDB) *jobutils.StateResolver {
	return jobutils.NewStateResolver(
		db.GetJob,
		db.GetJobState,
	)
}

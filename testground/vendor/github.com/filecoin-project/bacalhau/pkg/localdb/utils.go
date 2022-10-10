package localdb

import (
	jobutils "github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

func GetStateResolver(db LocalDB) *jobutils.StateResolver {
	return jobutils.NewStateResolver(
		db.GetJob,
		db.GetJobState,
	)
}

func EventFilterByType(eventType model.JobLocalEventType) LocalEventFilter {
	return func(ev model.JobLocalEvent) bool {
		return ev.EventName == eventType
	}
}

func EventFilterByTypeAndShard(eventType model.JobLocalEventType, shardIndex int) LocalEventFilter {
	return func(ev model.JobLocalEvent) bool {
		return ev.EventName == eventType && ev.ShardIndex == shardIndex
	}
}

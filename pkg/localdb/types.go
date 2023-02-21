package localdb

import (
	"context"

	model "github.com/filecoin-project/bacalhau/pkg/model/v1beta1"
)

type JobQuery struct {
	ID          string              `json:"id"`
	ClientID    string              `json:"clientID"`
	IncludeTags []model.IncludedTag `json:"include_tags"`
	ExcludeTags []model.ExcludedTag `json:"exclude_tags"`
	Limit       int                 `json:"limit"`
	Offset      int                 `json:"offset"`
	ReturnAll   bool                `json:"return_all"`
	SortBy      string              `json:"sort_by"`
	SortReverse bool                `json:"sort_reverse"`
}

type LocalEventFilter func(ev model.JobLocalEvent) bool

// A LocalDB will persist jobs and their state to the underlying storage.
// It also gives an efficient way to retrieve jobs using queries.
// The LocalDB is the local view of the world and the transport
// will get events to other nodes that will update their datastore.
//
// The LocalDB and Transport interfaces could be swapped out for some kind
// of smart contract implementation (e.g. FVM)
type LocalDB interface {
	GetJob(ctx context.Context, id string) (*model.Job, error)
	GetJobState(ctx context.Context, jobID string) (model.JobState, error)
	GetJobEvents(ctx context.Context, id string) ([]model.JobEvent, error)
	GetJobLocalEvents(ctx context.Context, id string) ([]model.JobLocalEvent, error)
	GetJobs(ctx context.Context, query JobQuery) ([]*model.Job, error)
	GetJobsCount(ctx context.Context, query JobQuery) (int, error)
	HasLocalEvent(ctx context.Context, jobID string, eventFilter LocalEventFilter) (bool, error)
	AddJob(ctx context.Context, j *model.Job) error
	AddEvent(ctx context.Context, jobID string, event model.JobEvent) error
	AddLocalEvent(ctx context.Context, jobID string, event model.JobLocalEvent) error
	UpdateJobDeal(ctx context.Context, jobID string, deal model.Deal) error
	UpdateShardState(
		ctx context.Context,
		jobID, nodeID string,
		shardIndex int,
		state model.JobShardState,
	) error
}

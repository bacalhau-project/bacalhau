package kvstore

import (
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/objectstore"
)

type ExecutionEnvelope struct {
	Execution store.Execution
}

func (e ExecutionEnvelope) OnUpdate() []objectstore.Indexer {
	return []objectstore.Indexer{
		objectstore.NewIndexer(
			PrefixJobs, e.Execution.Job.ID(), objectstore.AddToSetOperation(e.Execution.ID),
		),
	}
}

func (e ExecutionEnvelope) OnDelete() []objectstore.Indexer {
	return []objectstore.Indexer{
		objectstore.NewIndexer(
			PrefixJobs, e.Execution.Job.ID(), objectstore.DeleteFromSetOperation(e.Execution.ID),
		),
	}
}

type ExecutionHistoryEnvelope struct {
	History []store.ExecutionHistory
}

func (e ExecutionHistoryEnvelope) OnUpdate() []objectstore.Indexer { return nil }
func (e ExecutionHistoryEnvelope) OnDelete() []objectstore.Indexer { return nil }

type JobIndexEnvelope []string

func (j JobIndexEnvelope) OnUpdate() []objectstore.Indexer { return nil }
func (j JobIndexEnvelope) OnDelete() []objectstore.Indexer { return nil }

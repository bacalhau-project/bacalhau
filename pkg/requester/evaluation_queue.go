package requester

import (
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
)

type EvaluationQueue struct {
	jobStore jobstore.Store
	broker   orchestrator.EvaluationBroker
}

func NewEvaluationQueue(store jobstore.Store, broker orchestrator.EvaluationBroker) *EvaluationQueue {
	q := &EvaluationQueue{
		jobStore: store,
		broker:   broker,
	}

	return q
}

func (q *EvaluationQueue) Call() {

}

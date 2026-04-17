package evaluation

import (
	"container/heap"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/collections"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// compile-time assertion that evalWrapper satisfies the ScheduledTask interface
var _ collections.ScheduledTask[*models.Evaluation] = &evalWrapper{}

// evalWrapper satisfies the ScheduledTask interface
type evalWrapper struct {
	eval *models.Evaluation
}

func (d *evalWrapper) Data() *models.Evaluation {
	return d.eval
}

func (d *evalWrapper) ID() string {
	return d.eval.ID
}

func (d *evalWrapper) WaitUntil() time.Time {
	return d.eval.WaitUntil
}

// inflightEval tracks an unacknowledged evaluation along with the visibility timeout timer
type inflightEval struct {
	Eval            *models.Evaluation
	ReceiptHandle   string
	VisibilityTimer *time.Timer
}

// BrokerStats returns all the stats about the broker
type BrokerStats struct {
	TotalReady      int
	TotalInflight   int
	TotalPending    int
	TotalWaiting    int
	TotalCancelable int
	DelayedEvals    map[string]*models.Evaluation
	ByScheduler     map[string]*SchedulerStats
}

// IsEmpty returns true if the stats are zero
func (s *BrokerStats) IsEmpty() bool {
	if len(s.DelayedEvals) > 0 {
		return false
	}
	for _, v := range s.ByScheduler {
		if !v.IsEmpty() {
			return false
		}
	}
	return s.TotalReady == 0 &&
		s.TotalInflight == 0 &&
		s.TotalPending == 0 &&
		s.TotalWaiting == 0 &&
		s.TotalCancelable == 0
}

// SchedulerStats returns the stats per scheduler
type SchedulerStats struct {
	Ready    int
	Inflight int
}

// IsEmpty returns true if the scheduler stats are zero
func (s *SchedulerStats) IsEmpty() bool {
	return s.Ready == 0 && s.Inflight == 0
}

// ReadyEvaluations is a list of ready Evaluations across multiple jobs. We
// implement the container/heap interface so that this is a priority queue.
type ReadyEvaluations []*models.Evaluation

// Len is for the sorting interface
func (r ReadyEvaluations) Len() int {
	return len(r)
}

// Less is for the sorting interface. We flip the check
// so that the "min" in the min-heap is the element with the
// highest priority
func (r ReadyEvaluations) Less(i, j int) bool {
	if r[i].JobID != r[j].JobID && r[i].Priority != r[j].Priority {
		return r[i].Priority >= r[j].Priority
	}
	return r[i].CreateTime < r[j].CreateTime
}

// Swap is for the sorting interface
func (r ReadyEvaluations) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

// Push is used to add a new evaluation to the slice
func (r *ReadyEvaluations) Push(e interface{}) {
	*r = append(*r, e.(*models.Evaluation))
}

// Pop is used to remove an evaluation from the slice
func (r *ReadyEvaluations) Pop() interface{} {
	old := *r
	n := len(old)
	e := old[n-1]
	*r = old[:n-1]
	return e
}

// Peek is used to peek at the next element that would be popped
func (r ReadyEvaluations) Peek() *models.Evaluation {
	n := len(r)
	if n == 0 {
		return nil
	}
	return r[n-1]
}

// PendingEvaluations is a list of pending Evaluations for a given job. We
// implement the container/heap interface so that this is a priority queue.
type PendingEvaluations []*models.Evaluation

// Len is for the sorting interface
func (p PendingEvaluations) Len() int {
	return len(p)
}

// Less is for the sorting interface. We flip the check
// so that the "min" in the min-heap is the element with the
// highest priority or highest modify index
func (p PendingEvaluations) Less(i, j int) bool {
	if p[i].Priority != p[j].Priority {
		return p[i].Priority >= p[j].Priority
	}
	return p[i].ModifyTime >= p[j].ModifyTime
}

// Swap is for the sorting interface
func (p PendingEvaluations) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

// Push implements the heap interface and is used to add a new evaluation to the slice
func (p *PendingEvaluations) Push(e interface{}) {
	*p = append(*p, e.(*models.Evaluation))
}

// Pop implements the heap interface and is used to remove an evaluation from the slice
func (p *PendingEvaluations) Pop() interface{} {
	old := *p
	n := len(old)
	e := old[n-1]
	*p = old[:n-1]
	return e
}

// MarkForCancel is used to clear the pending list of all but the one with the
// highest modify index and highest priority. It returns a slice of cancelable
// evals so that Eval.Ack RPCs can write batched raft entries to cancel
// them. This must be called inside the broker's lock.
func (p *PendingEvaluations) MarkForCancel() []*models.Evaluation {
	// In pathological cases, we can have a large number of pending evals but
	// will want to cancel most of them. Using heap.Remove requires we re-sort
	// for each eval we remove. Because we expect to have at most one remaining,
	// we'll just create a new heap.
	retain := PendingEvaluations{(heap.Pop(p)).(*models.Evaluation)}

	cancelable := make([]*models.Evaluation, len(*p))
	copy(cancelable, *p)

	*p = retain
	return cancelable
}

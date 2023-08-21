//go:build unit || !integration

package evaluation

import (
	"container/heap"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/stretchr/testify/require"
)

func TestReadyEvals_Ordering(t *testing.T) {

	ready := ReadyEvaluations{}

	newEval := func(jobID, evalID string, priority int, index int64) *models.Evaluation {
		eval := mock.Eval()
		eval.JobID = jobID
		eval.ID = evalID
		eval.Priority = priority
		eval.CreateTime = index
		return eval
	}

	// note: we're intentionally pushing these out-of-order to assert we're
	// getting them back out in the intended order and not just as inserted
	heap.Push(&ready, newEval("example1", "eval01", 50, 1))
	heap.Push(&ready, newEval("example3", "eval03", 70, 3))
	heap.Push(&ready, newEval("example2", "eval02", 50, 2))

	next := heap.Pop(&ready).(*models.Evaluation)
	require.Equal(t, "eval03", next.ID,
		"expected highest Priority to be next ready")

	next = heap.Pop(&ready).(*models.Evaluation)
	require.Equal(t, "eval01", next.ID,
		"expected oldest CreateTime to be next ready")

	heap.Push(&ready, newEval("example4", "eval04", 50, 4))

	next = heap.Pop(&ready).(*models.Evaluation)
	require.Equal(t, "eval02", next.ID,
		"expected oldest CreateTime to be next ready")

}

func TestPendingEval_Ordering(t *testing.T) {
	pending := PendingEvaluations{}

	newEval := func(evalID string, priority int, index int64) *models.Evaluation {
		eval := mock.Eval()
		eval.ID = evalID
		eval.Priority = priority
		eval.ModifyTime = index
		return eval
	}

	// note: we're intentionally pushing these out-of-order to assert we're
	// getting them back out in the intended order and not just as inserted
	heap.Push(&pending, newEval("eval03", 50, 3))
	heap.Push(&pending, newEval("eval02", 100, 2))
	heap.Push(&pending, newEval("eval01", 50, 1))

	next := heap.Pop(&pending).(*models.Evaluation)
	require.Equal(t, "eval02", next.ID,
		"expected eval with highest priority to be next")

	next = heap.Pop(&pending).(*models.Evaluation)
	require.Equal(t, "eval03", next.ID,
		t, "expected eval with highest modify index to be next")

	heap.Push(&pending, newEval("eval04", 30, 4))
	next = heap.Pop(&pending).(*models.Evaluation)
	require.Equal(t, "eval01", next.ID,
		"expected eval with highest priority to be next")

}

func TestPendingEvals_MarkForCancel(t *testing.T) {
	pending := PendingEvaluations{}

	// note: we're intentionally pushing these out-of-order to assert we're
	// getting them back out in the intended order and not just as inserted
	for i := 100; i > 0; i -= 10 {
		eval := mock.Eval()
		eval.JobID = "example"
		eval.CreateTime = int64(i)
		eval.ModifyTime = int64(i)
		heap.Push(&pending, eval)
	}

	canceled := pending.MarkForCancel()
	require.Equal(t, 9, len(canceled))
	require.Equal(t, 1, pending.Len())

	raw := heap.Pop(&pending)
	require.NotNil(t, raw)
	eval := raw.(*models.Evaluation)
	require.EqualValues(t, 100, eval.ModifyTime)
}

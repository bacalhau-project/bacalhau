package collections

import (
	"container/heap"
	"errors"
	"sync"

	"github.com/samber/lo"
)

const (
	InitialQueueCapacity = 64
)

var (
	ErrEmptyQueue error = errors.New("queue is empty")
	ErrNoMatch    error = errors.New("no items matched")
)

type PriorityQueueInterface[T any] interface {
	// Enqueue will add the item specified by `data` to the queue with the
	// the priority given by `priority`.
	Enqueue(data T, priority int)

	// Dequeue returns the next highest priority item, returning both
	// the data Enqueued previously, and the priority with which it was
	// enqueued. An err (ErrEmptyQueue) may be returned if the queue is
	// currently empty.
	Dequeue() *QueueItem[T]

	// DequeueWhere allows the caller to iterate through the queue, in priority order, and
	// attempt to match an item using the provided `MatchingFunction`.  This method has a high
	// time cost as dequeued but non-matching items must be held and requeued once the process
	// is complete.  Luckily, we use the same amount of space (bar a few bytes for the
	// extra PriorityQueue) for the dequeued items.
	DequeueWhere(matcher MatchingFunction[T]) *QueueItem[T]

	// Len returns the number of items currently in the queue
	Len() int

	// IsEmpty returns a boolean denoting whether the queue is
	// currently empty or not.
	IsEmpty() bool
}

// PriorityQueue contains items of type T, and allows you to enqueue
// and dequeue items with a specific priority. Items are dequeued in
// highest priority first order.
type PriorityQueue[T any] struct {
	internalQueue queueHeap
	mu            sync.Mutex
}

// QueueItem encapsulates an item in the queue when we return it from
// the various dequeue methods
type QueueItem[T any] struct {
	Value    T
	Priority int
}

// MatchingFunction can be used when 'iterating' the priority queue to find
// items with specific properties.
type MatchingFunction[T any] func(possibleMatch T) bool

// NewPriorityQueue creates a new ptr to a priority queue for type T.
func NewPriorityQueue[T any]() *PriorityQueue[T] {
	q := &PriorityQueue[T]{
		internalQueue: make(queueHeap, 0, InitialQueueCapacity),
	}
	heap.Init(&q.internalQueue)
	return q
}

// Enqueue will add the item specified by `data` to the queue with the
// the priority given by `priority`.
func (pq *PriorityQueue[T]) Enqueue(data T, priority int) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	pq.enqueue(data, priority)
}

// enqueue is a lock-free version of Enqueue for internal use when a
// method already has a lock.
func (pq *PriorityQueue[T]) enqueue(data T, priority int) {
	heap.Push(
		&pq.internalQueue,
		&heapItem{
			value:    data,
			priority: priority,
		},
	)
}

// Dequeue returns the next highest priority item, returning both
// the data Enqueued previously, and the priority with which it was
// enqueued. An err (ErrEmptyQueue) may be returned if the queue is
// currently empty.
func (pq *PriorityQueue[T]) Dequeue() *QueueItem[T] {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	return pq.dequeue()
}

// dequeue is a lock-free version of Dequeue() for use by internal
// methods that already have a lock
func (pq *PriorityQueue[T]) dequeue() *QueueItem[T] {
	if pq.IsEmpty() {
		return nil
	}

	internalItem := heap.Pop(&pq.internalQueue)
	heapItem := internalItem.(*heapItem)
	item, _ := heapItem.value.(T)

	return &QueueItem[T]{Value: item, Priority: heapItem.priority}
}

// DequeueWhere allows the caller to iterate through the queue, in priority order, and
// attempt to match an item using the provided `MatchingFunction`.  This method has a high
// time cost as dequeued but non-matching items must be held and requeued once the process
// is complete.  Luckily, we use the same amount of space (bar a few bytes for the
// extra PriorityQueue) for the dequeued items.
func (pq *PriorityQueue[T]) DequeueWhere(matcher MatchingFunction[T]) *QueueItem[T] {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	var result *QueueItem[T] = nil

	// Create a new array to hold items that are not matches, this is suboptimal for time
	// but not really an issue for space as the items we add here will have been removed
	// from the queue.
	unmatched := make([]*QueueItem[T], 0, pq.Len())

	// Keep dequeueing items until one of them matches the function provided.
	// If any match it will be returned after the other items have been requeued.
	// If any iteration does not generate a match, the item is requeued in a temporary
	// queue reading for requeueing on this queue later on.
	for pq.internalQueue.Len() > 0 {
		qitem := pq.dequeue()

		if qitem == nil {
			return nil
		}

		if matcher(qitem.Value) {
			result = qitem
			break
		}

		// Add to the queue
		unmatched = append(unmatched, qitem)
	}

	// Re-add the items that were not matched back onto the Q
	lo.ForEach(unmatched, func(item *QueueItem[T], _ int) {
		pq.enqueue(item.Value, item.Priority)
	})

	// return the result we found, which might still be nil (not found)
	return result
}

// Len returns the number of items currently in the queue
func (pq *PriorityQueue[T]) Len() int {
	return pq.internalQueue.Len()
}

// IsEmpty returns a boolean denoting whether the queue is
// currently empty or not.
func (pq *PriorityQueue[T]) IsEmpty() bool {
	return pq.Len() == 0
}

// Internal priority queue implementation based on the go docs for
// `container/heap`. It is internal here to provide a generic and
// concurrent interface over the top of it.
type queueHeap []*heapItem

type heapItem struct {
	value    any
	priority int
	index    int // The index for update
}

func (q *queueHeap) Push(data any) {
	n := len(*q)
	item := data.(*heapItem)
	item.index = n
	*q = append(*q, item)
}

// Dequeue returns the highest priority item from the queue
func (q *queueHeap) Pop() any {
	old := *q
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*q = old[0 : n-1]
	return item
}

func (q queueHeap) Len() int { return len(q) }

func (q queueHeap) Less(i, j int) bool {
	// Dequeue returns highest priority so uses greater than here.
	return q[i].priority > q[j].priority
}

func (q queueHeap) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}

var _ PriorityQueueInterface[struct{}] = (*PriorityQueue[struct{}])(nil)

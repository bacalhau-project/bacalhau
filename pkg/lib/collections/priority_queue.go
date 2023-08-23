package collections

import (
	"container/heap"
	"errors"
	"sync"
)

const (
	InitialQueueCapacity = 64
)

var (
	ErrEmptyQueue error = errors.New("queue is empty")
	ErrNoMatch    error = errors.New("no items matched")
)

// PriorityQueue contains items of type T, and allows you to enqueue
// and dequeue items with a specific priority. Items are dequeued in
// highest priority first order.
type PriorityQueue[T any] struct {
	internalQueue queueHeap
	mu            sync.Mutex
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
func (pq *PriorityQueue[T]) Dequeue() (T, int, error) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	return pq.dequeue()
}

// dequeue is a lock-free version of Dequeue() for use by internal
// methods that already have a lock
func (pq *PriorityQueue[T]) dequeue() (T, int, error) {
	if pq.IsEmpty() {
		return *new(T), 0, ErrEmptyQueue
	}

	internalItem := heap.Pop(&pq.internalQueue)
	heapItem := internalItem.(*heapItem)
	item, ok := heapItem.value.(T)
	if !ok {
		// Something has gone very, very wrong if the item in the heap
		// is not a T as our Enqueue method should be the only thing
		// putting stuff into it.
		return item, 0, errors.New("priority queue found a bad type in the internal heap")
	}

	return item, heapItem.priority, nil
}

// DequeueWhere allows the caller to iterate through the queue, in priority order, and
// attempt to match an item using the provided `MatchingFunction`.  This method has a high
// time cost as dequeued but non-matching items must be held and requeued once the process
// is complete.  Luckily, we use the same amount of space (bar a few bytes for the
// extra PriorityQueue) for the dequeued items.
func (pq *PriorityQueue[T]) DequeueWhere(matcher MatchingFunction[T]) (T, error) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	var result T
	var found bool

	// Create a new Q to hold items that are not matches, this is suboptimal for time
	// but not really an issue for space as we'll be using the same.
	newQ := NewPriorityQueue[T]()

	// Keep dequeueing items until one of them matches the function provided.
	// If any match it will be returned after the other items have been requeued.
	// If any iteration does not generate a match, the item is requeued in a temporary
	// queue reading for requeueing on this queue later on.
	max := pq.internalQueue.Len()
	for i := 0; i < max; i++ {
		item, prio, err := pq.dequeue()
		if err != nil {
			return *new(T), err
		}

		if matcher(item) {
			result = item
			found = true
			break
		}

		// Add to the queue
		newQ.enqueue(item, prio)
	}

	// Re-add the items from newQ back onto the main queue after initializing
	// the new q so we can dequeue in priority order
	pq.Merge(newQ)

	if !found {
		return result, ErrNoMatch
	}

	return result, nil
}

func (pq *PriorityQueue[T]) Merge(other *PriorityQueue[T]) {
	heap.Init(&other.internalQueue)

	for {
		x, p, e := other.dequeue()
		if e != nil {
			break // break when the other queue is empty
		}

		pq.enqueue(x, p)
	}
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

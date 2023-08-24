package collections

import "sync"

type HashedPriorityQueue[K comparable, T any] struct {
	identifiers map[K]struct{}
	queue       *PriorityQueue[T]
	mu          sync.Mutex
	indexer     IndexerFunc[K, T]
}

// IndexerFunc is used to find the key (of type K) from the provided
// item (T). This will be used for the item lookup in `Contains`
type IndexerFunc[K comparable, T any] func(item T) K

// NewHashedPriorityQueue creates a new PriorityQueue that allows us to check if specific
// items (indexed by a key field) are present in the queue. The provided IndexerFunc will
// be used on Enqueue/Dequeue to keep the index up to date.
func NewHashedPriorityQueue[K comparable, T any](indexer IndexerFunc[K, T]) *HashedPriorityQueue[K, T] {
	return &HashedPriorityQueue[K, T]{
		identifiers: make(map[K]struct{}),
		queue:       NewPriorityQueue[T](),
		indexer:     indexer,
	}
}

// Contains will return true if the provided identifier (of type K)
// will be found in this queue, false if it is not present.
func (q *HashedPriorityQueue[K, T]) Contains(id K) bool {
	_, ok := q.identifiers[id]
	return ok
}

// Enqueue will add the item specified by `data` to the queue with the
// the priority given by `priority`.
func (q *HashedPriorityQueue[K, T]) Enqueue(data T, priority int) {
	q.mu.Lock()
	defer q.mu.Unlock()

	k := q.indexer(data)

	q.identifiers[k] = struct{}{}
	q.queue.Enqueue(data, priority)
}

// Dequeue returns the next highest priority item, returning both
// the data Enqueued previously, and the priority with which it was
// enqueued. An err (ErrEmptyQueue) may be returned if the queue is
// currently empty.
func (q *HashedPriorityQueue[K, T]) Dequeue() *QueueItem[T] {
	q.mu.Lock()
	defer q.mu.Unlock()

	item := q.queue.Dequeue()
	if item == nil {
		return nil
	}

	k := q.indexer(item.Value)
	delete(q.identifiers, k)

	return item
}

// DequeueWhere allows the caller to iterate through the queue, in priority order, and
// attempt to match an item using the provided `MatchingFunction`.  This method has a high
// time cost as dequeued but non-matching items must be held and requeued once the process
// is complete.  Luckily, we use the same amount of space (bar a few bytes for the
// extra PriorityQueue) for the dequeued items.
func (q *HashedPriorityQueue[K, T]) DequeueWhere(matcher MatchingFunction[T]) *QueueItem[T] {
	q.mu.Lock()
	defer q.mu.Unlock()

	item := q.queue.DequeueWhere(matcher)
	if item == nil {
		return nil
	}

	k := q.indexer(item.Value)
	delete(q.identifiers, k)

	return item
}

// Len returns the number of items currently in the queue
func (q *HashedPriorityQueue[K, T]) Len() int {
	return q.queue.Len()
}

// IsEmpty returns a boolean denoting whether the queue is
// currently empty or not.
func (q *HashedPriorityQueue[K, T]) IsEmpty() bool {
	return q.queue.Len() == 0
}

var _ PriorityQueueInterface[struct{}] = (*HashedPriorityQueue[string, struct{}])(nil)

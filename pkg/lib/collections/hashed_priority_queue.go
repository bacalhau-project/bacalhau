package collections

import (
	"sync"

	"github.com/rs/zerolog/log"
)

type HashedPriorityQueue[K comparable, T any] struct {
	identifiers map[K]struct{}
	queue       *PriorityQueue[T]
	mu          sync.RWMutex
	indexer     IndexerFunc[K, T]
}

// IndexerFunc is used to find the key (of type K) from the provided
// item (T). This will be used for the item lookup in `Contains`
type IndexerFunc[K comparable, T any] func(item T) K

// NewHashedPriorityQueue creates a new PriorityQueue that allows us to check if specific
// items (indexed by a key field) are present in the queue. The provided IndexerFunc will
// be used on Enqueue/Dequeue to keep the index up to date.
func NewHashedPriorityQueue[K comparable, T any](indexer IndexerFunc[K, T]) *HashedPriorityQueue[K, T] {
	log.Info().Msg("Creating new HashedPriorityQueue")
	return &HashedPriorityQueue[K, T]{
		identifiers: make(map[K]struct{}),
		queue:       NewPriorityQueue[T](),
		indexer:     indexer,
	}
}

// Contains will return true if the provided identifier (of type K)
// will be found in this queue, false if it is not present.
func (q *HashedPriorityQueue[K, T]) Contains(id K) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	_, ok := q.identifiers[id]
	log.Info().Bool("contains", ok).Interface("id", id).Msg("Checking if item is in the queue")
	return ok
}

// Enqueue will add the item specified by `data` to the queue with the
// the priority given by `priority`.
func (q *HashedPriorityQueue[K, T]) Enqueue(data T, priority int64) {
	q.mu.Lock()
	defer q.mu.Unlock()

	k := q.indexer(data)

	q.identifiers[k] = struct{}{}
	log.Info().Interface("key", k).Interface("data", data).Int64("priority", priority).Msg("Enqueuing item")
	q.queue.Enqueue(data, priority)

	log.Info().Int("queue_length", q.queue.Len()).Msg("Item enqueued, current queue length")
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
		log.Info().Msg("Queue is empty, no item to dequeue")
		return nil
	}

	// Find the key for the item and delete it from the presence map
	k := q.indexer(item.Value)
	delete(q.identifiers, k)

	log.Info().Interface("key", k).Interface("data", item.Value).Msg("Dequeued item")
	log.Info().Int("queue_length", q.queue.Len()).Msg("Item dequeued, current queue length")

	return item
}

// DequeueWhere allows the caller to iterate through the queue, in priority order, and
// attempt to match an item using the provided `MatchingFunction`. This method has a high
// time cost as dequeued but non-matching items must be held and requeued once the process
// is complete.
func (q *HashedPriorityQueue[K, T]) DequeueWhere(matcher MatchingFunction[T]) *QueueItem[T] {
	q.mu.Lock()
	defer q.mu.Unlock()

	item := q.queue.DequeueWhere(matcher)
	if item == nil {
		log.Info().Msg("No matching item found during DequeueWhere")
		return nil
	}

	k := q.indexer(item.Value)
	delete(q.identifiers, k)

	log.Info().Interface("key", k).Interface("data", item.Value).Msg("Dequeued matching item")
	return item
}

// Len returns the number of items currently in the queue
func (q *HashedPriorityQueue[K, T]) Len() int {
	length := q.queue.Len()
	log.Info().Int("queue_length", length).Msg("Current queue length")
	return length
}

// IsEmpty returns a boolean denoting whether the queue is
// currently empty or not.
func (q *HashedPriorityQueue[K, T]) IsEmpty() bool {
	empty := q.queue.Len() == 0
	log.Info().Bool("is_empty", empty).Msg("Checking if queue is empty")
	return empty
}

var _ PriorityQueueInterface[struct{}] = (*HashedPriorityQueue[string, struct{}])(nil)

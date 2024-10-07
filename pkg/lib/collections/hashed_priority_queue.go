package collections

import (
	"sync"
)

// HashedPriorityQueue is a priority queue that maintains only a single item per unique key.
// It combines the functionality of a hash map and a priority queue to provide efficient
// operations with the following key characteristics:
//
//  1. Single Item Per Key: The queue maintains only the latest version of an item for each
//     unique key. When a new item with an existing key is enqueued, it replaces the old item
//     instead of adding a duplicate.
//
//  2. Lazy Dequeuing: Outdated items (those that have been replaced by a newer version) are
//     not immediately removed from the underlying queue. Instead, they are filtered out
//     during dequeue operations. This approach improves enqueue performance
//     at the cost of potentially slower dequeue operations.
//
//  3. Higher Enqueue Throughput: By avoiding immediate removal of outdated items, the
//     HashedPriorityQueue achieves higher enqueue throughput. This makes it particularly
//     suitable for scenarios with frequent updates to existing items.
//
//  4. Eventually Consistent: The queue becomes consistent over time as outdated items are
//     lazily removed during dequeue operations. This means that the queue's length and the
//     items it contains become accurate as items are dequeued.
//
//  5. Memory Consideration: Due to the lazy removal of outdated items, the underlying queue
//     may temporarily hold more items than there are unique keys. This trade-off allows for
//     better performance but may use more memory compared to a strictly consistent queue.
//
// Use HashedPriorityQueue when you need a priority queue that efficiently handles updates
// to existing items and can tolerate some latency in removing outdated entries in favor
// of higher enqueue performance.
type HashedPriorityQueue[K comparable, T any] struct {
	identifiers map[K]int64
	queue       *PriorityQueue[versionedItem[T]]
	mu          sync.RWMutex
	indexer     IndexerFunc[K, T]
}

// versionedItem wraps the actual data item with a version number.
// This structure is used internally by HashedPriorityQueue to implement
// the versioning mechanism that allows for efficient updates and
// lazy removal of outdated items. The queue is only interested in
// the latest version of an item for each unique key:
//   - data: The actual item of type T stored in the queue.
//   - version: A monotonically increasing number representing the
//     version of this item. When an item with the same key is enqueued,
//     its version is incremented. This allows the queue to identify
//     the most recent version during dequeue operations and discard
//     any older versions of the same item.
type versionedItem[T any] struct {
	data    T
	version int64
}

// IndexerFunc is used to find the key (of type K) from the provided
// item (T). This will be used for the item lookup in `Contains`
type IndexerFunc[K comparable, T any] func(item T) K

// NewHashedPriorityQueue creates a new PriorityQueue that allows us to check if specific
// items (indexed by a key field) are present in the queue. The provided IndexerFunc will
// be used on Enqueue/Dequeue to keep the index up to date.
func NewHashedPriorityQueue[K comparable, T any](indexer IndexerFunc[K, T]) *HashedPriorityQueue[K, T] {
	return &HashedPriorityQueue[K, T]{
		identifiers: make(map[K]int64),
		queue:       NewPriorityQueue[versionedItem[T]](),
		indexer:     indexer,
	}
}

// isLatestVersion checks if the given item is the latest version
func (q *HashedPriorityQueue[K, T]) isLatestVersion(item versionedItem[T]) bool {
	k := q.indexer(item.data)
	currentVersion := q.identifiers[k]
	return item.version == currentVersion
}

// unwrapQueueItem converts a versionedItem to a QueueItem
func (q *HashedPriorityQueue[K, T]) unwrapQueueItem(item *QueueItem[versionedItem[T]]) *QueueItem[T] {
	return &QueueItem[T]{Value: item.Value.data, Priority: item.Priority}
}

// Contains will return true if the provided identifier (of type K)
// will be found in this queue, false if it is not present.
func (q *HashedPriorityQueue[K, T]) Contains(id K) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	_, ok := q.identifiers[id]
	return ok
}

// Enqueue will add the item specified by `data` to the queue with the
// the priority given by `priority`.
func (q *HashedPriorityQueue[K, T]) Enqueue(data T, priority int64) {
	q.mu.Lock()
	defer q.mu.Unlock()

	k := q.indexer(data)
	version := q.identifiers[k] + 1
	q.identifiers[k] = version
	q.queue.Enqueue(versionedItem[T]{data: data, version: version}, priority)
}

// Dequeue returns the next highest priority item, returning both
// the data Enqueued previously, and the priority with which it was
// enqueued. An err (ErrEmptyQueue) may be returned if the queue is
// currently empty.
func (q *HashedPriorityQueue[K, T]) Dequeue() *QueueItem[T] {
	q.mu.Lock()
	defer q.mu.Unlock()

	for {
		item := q.queue.Dequeue()
		if item == nil {
			return nil
		}

		if q.isLatestVersion(item.Value) {
			k := q.indexer(item.Value.data)
			delete(q.identifiers, k)
			return q.unwrapQueueItem(item)
		}
	}
}

// Peek returns the next highest priority item without removing it from the queue.
// It returns nil if the queue is empty.
func (q *HashedPriorityQueue[K, T]) Peek() *QueueItem[T] {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for {
		item := q.queue.Peek()
		if item == nil {
			return nil
		}

		if q.isLatestVersion(item.Value) {
			return q.unwrapQueueItem(item)
		}

		// If the peeked item is outdated, remove it and continue
		q.queue.Dequeue()
	}
}

// DequeueWhere allows the caller to iterate through the queue, in priority order, and
// attempt to match an item using the provided `MatchingFunction`.  This method has a high
// time cost as dequeued but non-matching items must be held and requeued once the process
// is complete.  Luckily, we use the same amount of space (bar a few bytes for the
// extra PriorityQueue) for the dequeued items.
func (q *HashedPriorityQueue[K, T]) DequeueWhere(matcher MatchingFunction[T]) *QueueItem[T] {
	q.mu.Lock()
	defer q.mu.Unlock()

	for {
		item := q.queue.DequeueWhere(func(vi versionedItem[T]) bool {
			return matcher(vi.data)
		})

		if item == nil {
			return nil
		}

		if q.isLatestVersion(item.Value) {
			k := q.indexer(item.Value.data)
			delete(q.identifiers, k)
			return q.unwrapQueueItem(item)
		}
	}
}

// Len returns the number of items currently in the queue
func (q *HashedPriorityQueue[K, T]) Len() int {
	return len(q.identifiers)
}

// IsEmpty returns a boolean denoting whether the queue is
// currently empty or not.
func (q *HashedPriorityQueue[K, T]) IsEmpty() bool {
	return q.Len() == 0
}

var _ PriorityQueueInterface[struct{}] = (*HashedPriorityQueue[string, struct{}])(nil)

package collections

import (
	"container/heap"
	"fmt"
	"time"
)

// ScheduledTaskHeap wraps a heap and provides deduplication and operations other than Push/Pop.
// The heap elements are sorted by the time in the WaitUntil field of scheduledHeapNode
type ScheduledTaskHeap[T any] struct {
	index map[string]*scheduledHeapNode[T]
	heap  scheduledHeapImpl[T]
}

// ScheduledTask is an interface type implemented by objects stored in the ScheduledTaskHeap
type ScheduledTask[T any] interface {
	Data() T              // The data object
	ID() string           // ID of the object
	WaitUntil() time.Time // Time to wait until
}

func NewScheduledTaskHeap[T any]() *ScheduledTaskHeap[T] {
	return &ScheduledTaskHeap[T]{
		index: make(map[string]*scheduledHeapNode[T]),
		heap:  make(scheduledHeapImpl[T], 0),
	}
}

func (h *ScheduledTaskHeap[T]) Push(task ScheduledTask[T]) error {
	if _, ok := h.index[task.ID()]; ok {
		return fmt.Errorf("task %s already exists", task.ID())
	}

	node := &scheduledHeapNode[T]{Task: task}
	h.index[task.ID()] = node
	heap.Push(&h.heap, node)
	return nil
}

func (h *ScheduledTaskHeap[T]) Pop() ScheduledTask[T] {
	if h.heap.Len() == 0 {
		return nil
	}

	node := heap.Pop(&h.heap).(*scheduledHeapNode[T])
	delete(h.index, node.Task.ID())
	return node.Task
}

func (h *ScheduledTaskHeap[T]) Peek() ScheduledTask[T] {
	if len(h.heap) == 0 {
		return nil
	}

	return h.heap[0].Task
}

func (h *ScheduledTaskHeap[T]) Contains(task ScheduledTask[T]) bool {
	_, ok := h.index[task.ID()]
	return ok
}

func (h *ScheduledTaskHeap[T]) Update(task ScheduledTask[T]) error {
	if existingNode, ok := h.index[task.ID()]; ok {
		existingNode.Task = task
		heap.Fix(&h.heap, existingNode.index)
		return nil
	}

	return fmt.Errorf("heap doesn't contain task with ID %q", task.ID())
}

func (h *ScheduledTaskHeap[T]) Remove(task ScheduledTask[T]) {
	if node, ok := h.index[task.ID()]; ok {
		heap.Remove(&h.heap, node.index)
		delete(h.index, task.ID())
	}
}

func (h *ScheduledTaskHeap[T]) Length() int {
	return h.heap.Len()
}

// scheduledHeapNode encapsulates the node stored in ScheduledTaskHeap
type scheduledHeapNode[T any] struct {
	// Task is the data object stored in the heap
	Task ScheduledTask[T]
	// index of the node in the heap, which is needed when adjusting the node's position
	// in the heap using heap.Fix
	index int
}

type scheduledHeapImpl[T any] []*scheduledHeapNode[T]

func (h scheduledHeapImpl[T]) Len() int {
	return len(h)
}

// Less sorts zero WaitUntil times at the end of the list, and normally
// otherwise
func (h scheduledHeapImpl[T]) Less(i, j int) bool {
	if h[i].Task.WaitUntil().IsZero() {
		return false
	}

	if h[j].Task.WaitUntil().IsZero() {
		return true
	}

	return h[i].Task.WaitUntil().Before(h[j].Task.WaitUntil())
}

func (h scheduledHeapImpl[T]) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *scheduledHeapImpl[T]) Push(x interface{}) {
	node := x.(*scheduledHeapNode[T])
	node.index = len(*h)
	*h = append(*h, node)
}

func (h *scheduledHeapImpl[T]) Pop() interface{} {
	old := *h
	node := old[h.Len()-1]
	node.index = -1 // for safety
	*h = old[0 : h.Len()-1]
	return node
}

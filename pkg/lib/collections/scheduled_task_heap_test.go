//go:build unit || !integration

package collections_test

import (
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/collections"
	"github.com/stretchr/testify/assert"
)

type mockTask struct {
	data      string
	id        string
	waitUntil time.Time
}

func (t *mockTask) Data() string {
	return t.data
}

func (t *mockTask) ID() string {
	return t.id
}

func (t *mockTask) WaitUntil() time.Time {
	return t.waitUntil
}

func TestScheduledTaskHeap_Push(t *testing.T) {
	h := collections.NewScheduledTaskHeap[string]()

	t1 := &mockTask{"task1", "1", time.Now().Add(10 * time.Minute)}

	assert.NoError(t, h.Push(t1))
	assert.Equal(t, 1, h.Length())
}

func TestScheduledTaskHeap_PushExistingID(t *testing.T) {
	h := collections.NewScheduledTaskHeap[string]()

	t1 := &mockTask{"task1", "1", time.Now().Add(10 * time.Minute)}
	t2 := &mockTask{"task2", "1", time.Now().Add(20 * time.Minute)}

	assert.NoError(t, h.Push(t1))
	assert.Equal(t, 1, h.Length())
	assert.Error(t, h.Push(t2))
	assert.Equal(t, 1, h.Length())
}

func TestScheduledTaskHeap_Pop(t *testing.T) {
	h := collections.NewScheduledTaskHeap[string]()

	t1 := &mockTask{"task1", "1", time.Now().Add(10 * time.Minute)}
	t2 := &mockTask{"task2", "2", time.Now().Add(5 * time.Minute)}

	assert.NoError(t, h.Push(t1))
	assert.NoError(t, h.Push(t2))

	assert.Equal(t, t2, h.Pop())
	assert.Equal(t, 1, h.Length())

	assert.Equal(t, t1, h.Pop())
	assert.Equal(t, 0, h.Length())
}

func TestScheduledTaskHeap_Peek(t *testing.T) {
	h := collections.NewScheduledTaskHeap[string]()

	t1 := &mockTask{"task1", "1", time.Now().Add(10 * time.Minute)}
	t2 := &mockTask{"task2", "2", time.Now().Add(5 * time.Minute)}

	assert.NoError(t, h.Push(t1))
	assert.NoError(t, h.Push(t2))

	assert.Equal(t, t2, h.Peek())
	assert.Equal(t, 2, h.Length())
}

func TestScheduledTaskHeap_Contains(t *testing.T) {
	h := collections.NewScheduledTaskHeap[string]()

	t1 := &mockTask{"task1", "1", time.Now().Add(10 * time.Minute)}
	t2 := &mockTask{"task2", "2", time.Now().Add(5 * time.Minute)}
	t3 := &mockTask{"task3", "3", time.Now().Add(15 * time.Minute)}

	assert.NoError(t, h.Push(t1))
	assert.NoError(t, h.Push(t2))

	assert.True(t, h.Contains(t1))
	assert.True(t, h.Contains(t2))
	assert.False(t, h.Contains(t3))
}

func TestScheduledTaskHeap_Update(t *testing.T) {
	h := collections.NewScheduledTaskHeap[string]()

	t1 := &mockTask{"task1", "1", time.Now().Add(10 * time.Minute)}

	assert.NoError(t, h.Push(t1))

	t1.waitUntil = time.Now().Add(5 * time.Minute)

	assert.NoError(t, h.Update(t1))
	assert.Equal(t, t1, h.Peek())
}

func TestScheduledTaskHeap_Delete(t *testing.T) {
	h := collections.NewScheduledTaskHeap[string]()

	t1 := &mockTask{"task1", "1", time.Now().Add(10 * time.Minute)}
	t2 := &mockTask{"task2", "2", time.Now().Add(5 * time.Minute)}

	assert.NoError(t, h.Push(t1))
	assert.NoError(t, h.Push(t2))

	h.Remove(t1)
	assert.Equal(t, 1, h.Length())

	assert.Equal(t, t2, h.Peek())
}

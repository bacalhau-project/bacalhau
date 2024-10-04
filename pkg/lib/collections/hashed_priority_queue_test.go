//go:build unit || !integration

package collections_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/lib/collections"
)

type HashedPriorityQueueSuite struct {
	suite.Suite
}

func TestHashedPriorityQueueSuite(t *testing.T) {
	suite.Run(t, new(HashedPriorityQueueSuite))
}

func (s *HashedPriorityQueueSuite) TestContains() {
	type TestData struct {
		id   string
		data int
	}

	indexer := func(t TestData) string {
		return t.id
	}

	q := collections.NewHashedPriorityQueue[string, TestData](indexer)

	s.Require().False(q.Contains("A"))
	q.Enqueue(TestData{id: "A", data: 0}, 1)
	s.Require().True(q.Contains("A"))
	_ = q.Dequeue()
	s.Require().False(q.Contains("A"))
}

func TestPriorityQueue_TimestampOrdering(t *testing.T) {
	type Item struct {
		ID       string
		Priority int64
	}

	pq := collections.NewPriorityQueue[Item]()

	// Enqueue items with different timestamps
	pq.Enqueue(Item{ID: "item1"}, 100)
	pq.Enqueue(Item{ID: "item2"}, 200)
	pq.Enqueue(Item{ID: "item3"}, 300)

	// Dequeue items and verify the order
	item := pq.Dequeue()
	assert.Equal(t, int64(100), item.Priority, "First item should have the lowest timestamp")

	item = pq.Dequeue()
	assert.Equal(t, int64(200), item.Priority, "Second item should have the next lowest timestamp")

	item = pq.Dequeue()
	assert.Equal(t, int64(300), item.Priority, "Third item should have the highest timestamp")
}

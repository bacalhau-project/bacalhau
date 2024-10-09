//go:build unit || !integration

package collections_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/lib/collections"
)

type HashedPriorityQueueSuite struct {
	PriorityQueueTestSuite
}

func (s *HashedPriorityQueueSuite) SetupTest() {
	s.NewQueue = func() collections.PriorityQueueInterface[TestData] {
		return collections.NewHashedPriorityQueue[string, TestData](func(t TestData) string {
			return t.id
		})
	}
}

func TestHashedPriorityQueueSuite(t *testing.T) {
	suite.Run(t, new(HashedPriorityQueueSuite))
}

func (s *HashedPriorityQueueSuite) TestContains() {
	q := s.NewQueue().(*collections.HashedPriorityQueue[string, TestData])

	s.Require().False(q.Contains("A"))
	q.Enqueue(TestData{"A", 0}, 1)
	s.Require().True(q.Contains("A"))
	_ = q.Dequeue()
	s.Require().False(q.Contains("A"))
}

func (s *HashedPriorityQueueSuite) TestPeek() {
	q := s.NewQueue().(*collections.HashedPriorityQueue[string, TestData])

	q.Enqueue(TestData{"A", 1}, 1)
	q.Enqueue(TestData{"B", 2}, 2)

	item := q.Peek()
	s.Require().NotNil(item)
	s.Require().Equal(TestData{"B", 2}, item.Value)
	s.Require().True(q.Contains("A"), "Item A should still be in the queue after Peek")
	s.Require().True(q.Contains("B"), "Item B should still be in the queue after Peek")

	_ = q.Dequeue()
	s.Require().False(q.Contains("B"), "Item B should not be in the queue after Dequeue")
	s.Require().True(q.Contains("A"), "Item A should still be in the queue after Dequeue")
}

func (s *HashedPriorityQueueSuite) TestSingleItemPerKey() {
	q := s.NewQueue().(*collections.HashedPriorityQueue[string, TestData])

	q.Enqueue(TestData{"A", 1}, 1)
	q.Enqueue(TestData{"A", 2}, 2)
	q.Enqueue(TestData{"A", 3}, 3)

	s.Require().Equal(1, q.Len(), "Queue should only contain one item for key 'A'")

	item := q.Dequeue()
	s.Require().NotNil(item)
	s.Require().Equal(TestData{"A", 3}, item.Value, "Should return the latest version of item 'A'")
	s.Require().Equal(int64(3), item.Priority, "Should have the priority of the latest enqueue")

	s.Require().Nil(q.Dequeue(), "Queue should be empty after dequeuing the single item")
}

func (s *HashedPriorityQueueSuite) TestPeekReturnsLatestVersion() {
	q := s.NewQueue().(*collections.HashedPriorityQueue[string, TestData])

	q.Enqueue(TestData{"A", 1}, 1)
	q.Enqueue(TestData{"B", 1}, 3)
	q.Enqueue(TestData{"A", 2}, 2)

	item := q.Peek()
	s.Require().NotNil(item)
	s.Require().Equal(TestData{"B", 1}, item.Value, "Peek should return 'B' as it has the highest priority")
	s.Require().Equal(int64(3), item.Priority)

	q.Enqueue(TestData{"B", 2}, 1) // Lower priority, but newer version

	item = q.Peek()
	s.Require().NotNil(item)
	s.Require().Equal(TestData{"A", 2}, item.Value, "Peek should now return 'A' as 'B' has lower priority")
	s.Require().Equal(int64(2), item.Priority)
}

func (s *HashedPriorityQueueSuite) TestDequeueWhereReturnsLatestVersion() {
	q := s.NewQueue().(*collections.HashedPriorityQueue[string, TestData])

	q.Enqueue(TestData{"A", 1}, 1)
	q.Enqueue(TestData{"B", 1}, 2)
	q.Enqueue(TestData{"A", 2}, 3)

	item := q.DequeueWhere(func(td TestData) bool {
		return td.id == "A"
	})

	s.Require().NotNil(item)
	s.Require().Equal(TestData{"A", 2}, item.Value, "DequeueWhere should return the latest version of 'A'")
	s.Require().Equal(int64(3), item.Priority)

	s.Require().False(q.Contains("A"), "A should no longer be in the queue")
	s.Require().True(q.Contains("B"), "B should still be in the queue")
}

func (s *HashedPriorityQueueSuite) TestDuplicateKeys() {
	inputs := []struct {
		v TestData
		p int64
	}{
		{TestData{"A", 1}, 3},
		{TestData{"B", 2}, 2},
		{TestData{"A", 3}, 1}, // Duplicate key with lower priority
		{TestData{"C", 4}, 4},
		{TestData{"B", 5}, 5}, // Duplicate key with higher priority
	}

	pq := s.NewQueue()
	for _, tc := range inputs {
		pq.Enqueue(tc.v, tc.p)
	}

	expected := []struct {
		v TestData
		p int64
	}{
		{TestData{"B", 5}, 5},
		{TestData{"C", 4}, 4},
		{TestData{"A", 3}, 1},
	}

	for _, exp := range expected {
		qitem := pq.Dequeue()
		s.Require().NotNil(qitem)
		s.Require().Equal(exp.v, qitem.Value)
		s.Require().Equal(exp.p, qitem.Priority)
	}

	s.Require().True(pq.IsEmpty())
}

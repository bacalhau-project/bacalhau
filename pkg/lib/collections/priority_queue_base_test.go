//go:build unit || !integration

package collections_test

import (
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/lib/collections"
)

type TestData struct {
	id   string
	data int
}

type PriorityQueueTestSuite struct {
	suite.Suite
	NewQueue func() collections.PriorityQueueInterface[TestData]
}

func (s *PriorityQueueTestSuite) TestSimple() {
	type testcase struct {
		v TestData
		p int64
	}
	inputs := []testcase{
		{TestData{"B", 2}, 2}, {TestData{"A", 1}, 3}, {TestData{"C", 3}, 1},
	}
	expected := []testcase{
		{TestData{"A", 1}, 3}, {TestData{"B", 2}, 2}, {TestData{"C", 3}, 1},
	}

	pq := s.NewQueue()
	for _, tc := range inputs {
		pq.Enqueue(tc.v, tc.p)
	}

	for _, tc := range expected {
		qItem := pq.Dequeue()
		s.Require().NotNil(qItem)
		s.Require().Equal(tc.v, qItem.Value)
		s.Require().Equal(tc.p, qItem.Priority)
	}

	s.Require().True(pq.IsEmpty())
}

func (s *PriorityQueueTestSuite) TestSimpleMin() {
	type testcase struct {
		v TestData
		p int64
	}
	inputs := []testcase{
		{TestData{"B", 2}, -2}, {TestData{"A", 1}, -3}, {TestData{"C", 3}, -1},
	}
	expected := []testcase{
		{TestData{"C", 3}, -1}, {TestData{"B", 2}, -2}, {TestData{"A", 1}, -3},
	}

	pq := s.NewQueue()
	for _, tc := range inputs {
		pq.Enqueue(tc.v, tc.p)
	}

	for _, tc := range expected {
		qItem := pq.Dequeue()
		s.Require().NotNil(qItem)
		s.Require().Equal(tc.v, qItem.Value)
		s.Require().Equal(tc.p, qItem.Priority)
	}

	s.Require().True(pq.IsEmpty())
}

func (s *PriorityQueueTestSuite) TestEmpty() {
	pq := s.NewQueue()
	qItem := pq.Dequeue()
	s.Require().Nil(qItem)
	s.Require().True(pq.IsEmpty())
}

func (s *PriorityQueueTestSuite) TestDequeueWhere() {
	pq := s.NewQueue()
	pq.Enqueue(TestData{"A", 1}, 4)
	pq.Enqueue(TestData{"D", 4}, 1)
	pq.Enqueue(TestData{"D", 4}, 1)
	pq.Enqueue(TestData{"D", 4}, 1)
	pq.Enqueue(TestData{"D", 4}, 1)
	pq.Enqueue(TestData{"B", 2}, 3)
	pq.Enqueue(TestData{"C", 3}, 2)

	count := pq.Len()

	qItem := pq.DequeueWhere(func(possibleMatch TestData) bool {
		return possibleMatch.id == "B"
	})

	s.Require().NotNil(qItem)
	s.Require().Equal(TestData{"B", 2}, qItem.Value)
	s.Require().Equal(int64(3), qItem.Priority)
	s.Require().Equal(count-1, pq.Len())
}

func (s *PriorityQueueTestSuite) TestDequeueWhereFail() {
	pq := s.NewQueue()
	pq.Enqueue(TestData{"A", 1}, 4)

	qItem := pq.DequeueWhere(func(possibleMatch TestData) bool {
		return possibleMatch.id == "Z"
	})

	s.Require().Nil(qItem)
}

func (s *PriorityQueueTestSuite) TestPeek() {
	pq := s.NewQueue()

	// Test 1: Peek on an empty queue
	item := pq.Peek()
	s.Require().Nil(item, "Peek on an empty queue should return nil")

	// Test 2: Peek after adding one item
	pq.Enqueue(TestData{"A", 1}, 1)
	item = pq.Peek()
	s.Require().NotNil(item, "Peek should return an item")
	s.Require().Equal(TestData{"A", 1}, item.Value, "Peek should return the correct value")
	s.Require().Equal(int64(1), item.Priority, "Peek should return the correct priority")
	s.Require().Equal(1, pq.Len(), "Peek should not remove the item from the queue")

	// Test 3: Peek with multiple items
	pq.Enqueue(TestData{"B", 2}, 3)
	pq.Enqueue(TestData{"C", 3}, 2)
	item = pq.Peek()
	s.Require().NotNil(item, "Peek should return an item")
	s.Require().Equal(TestData{"B", 2}, item.Value, "Peek should return the highest priority item")
	s.Require().Equal(int64(3), item.Priority, "Peek should return the correct priority")
	s.Require().Equal(3, pq.Len(), "Peek should not remove any items from the queue")

	// Test 4: Peek after dequeue
	dequeuedItem := pq.Dequeue()
	s.Require().Equal(TestData{"B", 2}, dequeuedItem.Value, "Dequeue should return the highest priority item")
	item = pq.Peek()
	s.Require().NotNil(item, "Peek should return an item")
	s.Require().Equal(TestData{"C", 3}, item.Value, "Peek should return the new highest priority item after dequeue")
	s.Require().Equal(int64(2), item.Priority, "Peek should return the correct priority")
	s.Require().Equal(2, pq.Len(), "Queue length should be reduced after dequeue")

	// Test 5: Multiple peeks should return the same item
	item1 := pq.Peek()
	item2 := pq.Peek()
	s.Require().Equal(item1, item2, "Multiple peeks should return the same item")
	s.Require().Equal(2, pq.Len(), "Multiple peeks should not change the queue length")
}

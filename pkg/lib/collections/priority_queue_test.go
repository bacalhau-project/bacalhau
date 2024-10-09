//go:build unit || !integration

package collections_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/lib/collections"
)

type PriorityQueueSuite struct {
	PriorityQueueTestSuite
}

func (s *PriorityQueueSuite) SetupTest() {
	s.NewQueue = func() collections.PriorityQueueInterface[TestData] {
		return collections.NewPriorityQueue[TestData]()
	}
}

func TestPriorityQueueSuite(t *testing.T) {
	suite.Run(t, new(PriorityQueueSuite))
}

func (s *PriorityQueueSuite) TestDuplicateKeys() {
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
		{TestData{"A", 1}, 3},
		{TestData{"B", 2}, 2},
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

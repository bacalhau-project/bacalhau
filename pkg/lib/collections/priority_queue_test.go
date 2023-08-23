//go:build unit || !integration

package collections_test

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/lib/collections"
	"github.com/stretchr/testify/suite"
)

type PriorityQueueSuite struct {
	suite.Suite
}

func TestPriorityQueueSuite(t *testing.T) {
	suite.Run(t, new(PriorityQueueSuite))
}

func (s *PriorityQueueSuite) TestSimple() {
	type testcase struct {
		v string
		p int
	}
	testcases := []testcase{
		{"A", 3},
		{"A", 3},
		{"B", 2},
		{"C", 1},
		{"C", 1},
	}

	pq := collections.NewPriorityQueue[string]()
	for _, tc := range testcases {
		pq.Enqueue(tc.v, tc.p)
	}

	for _, tc := range testcases {
		v, p, e := pq.Dequeue()
		s.Require().NoError(e)
		s.Require().Equal(tc.v, v)
		s.Require().Equal(tc.p, p)
	}

	s.Require().True(pq.IsEmpty())
}

func (s *PriorityQueueSuite) TestEmpty() {
	pq := collections.NewPriorityQueue[string]()
	_, p, e := pq.Dequeue()
	s.Require().Error(e)
	s.Require().ErrorIs(e, collections.ErrEmptyQueue)
	s.Require().Zero(p)
	s.Require().True(pq.IsEmpty())
}

func (s *PriorityQueueSuite) TestDequeueWhere() {
	pq := collections.NewPriorityQueue[string]()
	pq.Enqueue("A", 4)
	pq.Enqueue("D", 1)
	pq.Enqueue("D", 1)
	pq.Enqueue("D", 1)
	pq.Enqueue("D", 1)
	pq.Enqueue("B", 3)
	pq.Enqueue("C", 2)

	count := pq.Len()

	item, prio, err := pq.DequeueWhere(func(possibleMatch string) bool {
		return possibleMatch == "B"
	})

	s.Require().NoError(err)
	s.Require().Equal("B", item)
	s.Require().Equal(3, prio)
	s.Require().Equal(count-1, pq.Len())

}

func (s *PriorityQueueSuite) TestDequeueWhereFail() {
	pq := collections.NewPriorityQueue[string]()
	pq.Enqueue("A", 4)

	_, _, err := pq.DequeueWhere(func(possibleMatch string) bool {
		return possibleMatch == "Z"
	})

	s.Require().Error(err)
}

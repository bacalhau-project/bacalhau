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
		p int64
	}
	inputs := []testcase{
		{"B", 2}, {"A", 3}, {"C", 1}, {"A", 3}, {"C", 1}, {"B", 2},
	}
	expected := []testcase{
		{"A", 3}, {"A", 3}, {"B", 2}, {"B", 2}, {"C", 1}, {"C", 1},
	}

	pq := collections.NewPriorityQueue[string]()
	for _, tc := range inputs {
		pq.Enqueue(tc.v, int64(tc.p))
	}

	for _, tc := range expected {
		qitem := pq.Dequeue()
		s.Require().NotNil(qitem)
		s.Require().Equal(tc.v, qitem.Value)
		s.Require().Equal(tc.p, qitem.Priority)
	}

	s.Require().True(pq.IsEmpty())
}

func (s *PriorityQueueSuite) TestSimpleMin() {
	type testcase struct {
		v string
		p int64
	}
	inputs := []testcase{
		{"B", -2}, {"A", -3}, {"C", -1}, {"A", -3}, {"C", -1}, {"B", -2},
	}
	expected := []testcase{
		{"C", -1}, {"C", -1}, {"B", -2}, {"B", -2}, {"A", -3}, {"A", -3},
	}

	pq := collections.NewPriorityQueue[string]()
	for _, tc := range inputs {
		pq.Enqueue(tc.v, int64(tc.p))
	}

	for _, tc := range expected {
		qitem := pq.Dequeue()
		s.Require().NotNil(qitem)
		s.Require().Equal(tc.v, qitem.Value)
		s.Require().Equal(tc.p, qitem.Priority)
	}

	s.Require().True(pq.IsEmpty())
}

func (s *PriorityQueueSuite) TestEmpty() {
	pq := collections.NewPriorityQueue[string]()
	qitem := pq.Dequeue()
	s.Require().Nil(qitem)
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

	qitem := pq.DequeueWhere(func(possibleMatch string) bool {
		return possibleMatch == "B"
	})

	s.Require().NotNil(qitem)
	s.Require().Equal("B", qitem.Value)
	s.Require().Equal(int64(3), qitem.Priority)
	s.Require().Equal(count-1, pq.Len())

}

func (s *PriorityQueueSuite) TestDequeueWhereFail() {
	pq := collections.NewPriorityQueue[string]()
	pq.Enqueue("A", 4)

	qitem := pq.DequeueWhere(func(possibleMatch string) bool {
		return possibleMatch == "Z"
	})

	s.Require().Nil(qitem)
}

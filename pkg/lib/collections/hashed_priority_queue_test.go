//go:build unit || !integration

package collections_test

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/lib/collections"
	"github.com/stretchr/testify/suite"
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

//go:build unit || !integration

package queue_test

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/objectstore/distributed"
	"github.com/bacalhau-project/bacalhau/pkg/objectstore/queue"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type QueueTestSuite struct {
	suite.Suite
	ctx   context.Context
	store *distributed.DistributedObjectStore
}

func TestQueueTestSuite(t *testing.T) {
	suite.Run(t, &QueueTestSuite{
		ctx: context.Background(),
	})
}

func (s *QueueTestSuite) SetupTest() {
	s.store, _ = distributed.New(distributed.WithTestConfig())
}

func (s *QueueTestSuite) TearDownTest() {
	s.store.Close(s.ctx)
}

func (s *QueueTestSuite) TestSimple() {
	type testdata struct {
		Name string
	}

	q := queue.NewQueue[testdata](s.store, "testq")
	err := q.Enqueue(s.ctx, testdata{Name: "test"}, queue.QueuePriority(1))
	require.NoError(s.T(), err)

	t, err := q.Dequeue(s.ctx)
	require.NoError(s.T(), err)
	require.Equal(s.T(), "test", t.Name)

	q.Close()
}

func (s *QueueTestSuite) TestSimpleWithWait() {
	type testdata struct {
		Name string
	}

	tc1 := testdata{Name: "First item"}
	tc2 := testdata{Name: "Second item"}

	q := queue.NewQueue[testdata](s.store, "testq")

	// Enqueue tc1
	err := q.Enqueue(s.ctx, tc1, queue.QueuePriority(1))
	require.NoError(s.T(), err)

	// Dequeue tc1
	t, err := q.Dequeue(s.ctx)
	require.NoError(s.T(), err)
	require.Equal(s.T(), "First item", t.Name)

	doneChan := make(chan struct{})

	go func() {
		defer func() { doneChan <- struct{}{} }()

		t, err = q.Dequeue(s.ctx) // should block until we put

		require.NoError(s.T(), err)
		require.Equal(s.T(), tc2.Name, t.Name)
	}()

	_ = q.Enqueue(s.ctx, tc2, queue.QueuePriority(1))

	<-doneChan
	q.Close()
}

func (s *QueueTestSuite) TestPriorities() {
	type testdata struct {
		Name string
		P    uint8
	}

	tc1 := testdata{Name: "First item", P: 2}
	tc2 := testdata{Name: "Second item", P: 1}

	q := queue.NewQueue[testdata](s.store, "testq")

	// Enqueue tc1 with lower priority than tc2
	err := q.Enqueue(s.ctx, tc1, queue.QueuePriority(tc1.P))
	require.NoError(s.T(), err)

	err = q.Enqueue(s.ctx, tc2, queue.QueuePriority(tc2.P))
	require.NoError(s.T(), err)

	// Dequeue tc1
	t, err := q.Dequeue(s.ctx)
	require.NoError(s.T(), err)
	require.Equal(s.T(), "Second item", t.Name)

	t, err = q.Dequeue(s.ctx)
	require.NoError(s.T(), err)
	require.Equal(s.T(), "First item", t.Name)

	q.Close()
}

//go:build unit || !integration

package generic_test

import (
	"testing"
	"time"

	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RingBufferTestSuite struct {
	suite.Suite
}

func TestRingBufferTestSuite(t *testing.T) {
	suite.Run(t, new(RingBufferTestSuite))
}

func (s *RingBufferTestSuite) TestRingBufferSimple() {
	rb := generic.NewRingBuffer[int](10)

	for i := 0; i < 10; i++ {
		rb.Enqueue(i)
	}

	for i := 0; i < 10; i++ {
		x := rb.Dequeue()
		require.Equal(s.T(), i, x)
	}
}

func (s *RingBufferTestSuite) TestRingBufferCycle() {
	rb := generic.NewRingBuffer[int](3)

	for i := 0; i < 3; i++ {
		rb.Enqueue(i)
	}

	for i := 0; i < 9; i++ {
		x := rb.Dequeue()
		require.Equal(s.T(), i%3, x)
	}
}

func (s *RingBufferTestSuite) TestRingBufferBlock() {
	rb := generic.NewRingBuffer[int](3)

	go func() {
		x := rb.Dequeue()
		require.Equal(s.T(), 1, x)
	}()

	time.Sleep(time.Duration(100) * time.Millisecond)
	rb.Enqueue(1)
}

func (s *RingBufferTestSuite) TestRingBufferOverwrite() {
	rb := generic.NewRingBuffer[string](3)

	done := make(chan bool, 1)

	go func() {
		// Dequeue 3 items, then
		for i := 0; i < 3; i++ {
			d := rb.Dequeue()
			require.NotNil(s.T(), d)
			require.Equal(s.T(), "message", d)
		}
		done <- true
	}()

	for i := 0; i < 5; i++ {
		rb.Enqueue("message")
	}
	<-done

	// Ensure that we haven't expanded the ring at all
	count := 0
	rb.Each(func(r any) { count += 1 })
	require.Equal(s.T(), 3, count)
}

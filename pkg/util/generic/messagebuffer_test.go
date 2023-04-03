//go:build unit || !integration

package generic_test

import (
	"testing"

	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type MessageBufferTestSuite struct {
	suite.Suite
}

func TestMessageBufferTestSuite(t *testing.T) {
	suite.Run(t, new(MessageBufferTestSuite))
}

type datatype struct{ data string }

func (d datatype) GetDataSize() int64 {
	return int64(len(d.data))
}

func (s *MessageBufferTestSuite) TestMessageBufferSimple() {
	rb := generic.NewMessageBuffer[datatype](10, 1024)

	for i := 0; i < 10; i++ {
		d := datatype{data: "message"}
		rb.Enqueue(&d)
	}

	// Should be able to dequeue 10 items
	for i := 0; i < 10; i++ {
		d, err := rb.Dequeue()
		require.NoError(s.T(), err)
		require.NotNil(s.T(), d)
		require.Equal(s.T(), "message", d.data)
	}
}

func (s *MessageBufferTestSuite) TestMessageBufferOverflow() {
	rb := generic.NewMessageBuffer[datatype](3, 21)

	// We should only manage 3 items, and we should
	// drop 2 (as our string is 7 bytes and max is 21)

	for i := 0; i < 5; i++ {
		d := datatype{data: "message"}
		rb.Enqueue(&d)
	}

	// Should be able to dequeue 3 items
	for i := 0; i < 3; i++ {
		d, err := rb.Dequeue()
		require.NoError(s.T(), err)
		require.NotNil(s.T(), d)
		require.Equal(s.T(), "message", d.data)
	}

	drained := rb.Drain()
	require.Equal(s.T(), 0, len(drained), "queue should have been empty")
}

func (s *MessageBufferTestSuite) TestMessageBufferFlow() {
	rb := generic.NewMessageBuffer[datatype](3, 100)

	done := make(chan bool, 1)

	go func() {
		// Dequeue 3 items, then
		for i := 0; i < 3; i++ {
			d, err := rb.Dequeue()
			require.NoError(s.T(), err)
			require.NotNil(s.T(), d)
			require.Equal(s.T(), "message", d.data)
		}
		done <- true
	}()

	for i := 0; i < 5; i++ {
		d := datatype{data: "message"}
		rb.Enqueue(&d)
	}
	<-done
	rb.Close()

	drained := rb.Drain()
	require.Equal(s.T(), 2, len(drained), "queue should have had 2 items")
}

package pubsub

import (
	"context"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/suite"
)

type BufferingPubSubSuite struct {
	suite.Suite
	pusSub         *BufferingPubSub[string]
	subscriber     *InMemorySubscriber[string]
	cleanupManager *system.CleanupManager
	maxBufferSize  int64
	maxBufferAge   time.Duration
}

func (s *BufferingPubSubSuite) SetupTest() {
	s.maxBufferSize = 10
	s.maxBufferAge = 1 * time.Minute
	s.cleanupManager = system.NewCleanupManager()
	s.setupBuffer()

}

func (s *BufferingPubSubSuite) setupBuffer() {
	s.pusSub = NewBufferingPubSub[string](s.cleanupManager, BufferingPubSubParams{
		DelegatePubSub: NewInMemoryPubSub[BufferingEnvelope](),
		MaxBufferAge:   s.maxBufferAge,
		MaxBufferSize:  s.maxBufferSize,
	})
	s.subscriber = NewInMemorySubscriber[string]()
	s.pusSub.Subscribe(context.Background(), s.subscriber)
}

func TestBufferingPubSubSuite(t *testing.T) {
	suite.Run(t, new(BufferingPubSubSuite))
}

func (s *BufferingPubSubSuite) TestBufferingPubSub() {
	ctx := context.Background()
	// Every character results in 3 json bytes. We need 4 characters to reach the max buffer size of 10 bytes
	toWrite := []string{"a", "b", "c", "d"}

	for _, message := range toWrite[:3] {
		s.NoError(s.pusSub.Publish(ctx, message))
		s.Empty(s.subscriber.Events())
		s.True(s.pusSub.currentBuffer.Size() < s.maxBufferSize)
	}

	err := s.pusSub.Publish(ctx, toWrite[3])
	s.NoError(err)
	s.Equal(toWrite, s.subscriber.Events())
	s.Equal(int64(0), s.pusSub.currentBuffer.Size())
}

func (s *BufferingPubSubSuite) TestBufferingPubSub_SingleLargeMessage() {
	message := "1234567890 abcdefghijklmnopqrstuvwxyz"
	s.NoError(s.pusSub.Publish(context.Background(), message))
	s.Equal([]string{message}, s.subscriber.Events())
}

// Verify buffer is reset after each flush
func (s *BufferingPubSubSuite) TestBufferingPubSub_Reset() {
	ctx := context.Background()
	toWrite1 := []string{"a", "b", "c", "d"}
	toWrite2 := []string{"e", "f", "g", "h"}
	for _, message := range toWrite1 {
		s.NoError(s.pusSub.Publish(ctx, message))
	}
	s.Equal(toWrite1, s.subscriber.Events())

	// Verify messages from previous buffer don't show up in the new buffer
	for _, message := range toWrite2 {
		s.NoError(s.pusSub.Publish(ctx, message))
	}
	s.Equal(toWrite2, s.subscriber.Events())
}

func (s *BufferingPubSubSuite) TestBufferingPubSub_MaxBufferAge() {
	ctx := context.Background()
	s.maxBufferAge = 500 * time.Millisecond
	s.setupBuffer()

	s.NoError(s.pusSub.Publish(ctx, "a"))
	s.Empty(s.subscriber.Events())

	time.Sleep(s.maxBufferAge * 2)
	s.Equal([]string{"a"}, s.subscriber.Events())
}

func (s *BufferingPubSubSuite) TestBufferingPubSub_MaxBufferAge_Empty() {
	// verify nothing was flushed if the buffer is empty
	s.maxBufferAge = 200 * time.Millisecond
	s.setupBuffer()
	s.Empty(s.subscriber.Events())

	time.Sleep(s.maxBufferAge * 2)
	s.Empty(s.subscriber.Events())
}

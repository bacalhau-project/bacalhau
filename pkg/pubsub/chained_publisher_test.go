//go:build unit || !integration

package pubsub

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ChainedPublisherSuite struct {
	suite.Suite
	chain        *ChainedPublisher[string]
	ignoreErrors bool
	publisher1   *InMemoryPubSub[string]
	publisher2   *InMemoryPubSub[string]
	subscriber1  *InMemorySubscriber[string]
	subscriber2  *InMemorySubscriber[string]
}

func (s *ChainedPublisherSuite) SetupTest() {
	ctx := context.Background()
	s.publisher1 = NewInMemoryPubSub[string]()
	s.publisher2 = NewInMemoryPubSub[string]()
	s.subscriber1 = NewInMemorySubscriber[string]()
	s.subscriber2 = NewInMemorySubscriber[string]()

	s.Require().NoError(s.publisher1.Subscribe(ctx, s.subscriber1))
	s.Require().NoError(s.publisher2.Subscribe(ctx, s.subscriber2))
}

func (s *ChainedPublisherSuite) setupChain() {
	s.chain = NewChainedPublisher[string](s.ignoreErrors)
	s.chain.Add(s.publisher1)
	s.chain.Add(s.publisher2)
}

func TestChainedPublisherSuite(t *testing.T) {
	suite.Run(t, new(ChainedPublisherSuite))
}

func (s *ChainedPublisherSuite) TestPublish_NoErrors() {
	s.ignoreErrors = false
	s.setupChain()

	ctx := context.Background()
	message := "test"
	s.NoError(s.chain.Publish(ctx, message))
	s.Equal([]string{message}, s.subscriber1.Events())
	s.Equal([]string{message}, s.subscriber2.Events())
}

func (s *ChainedPublisherSuite) TestPublish_IgnoreErrors() {
	s.ignoreErrors = true
	s.subscriber1.badSubscriber = true
	s.setupChain()

	ctx := context.Background()
	message := "test"
	s.NoError(s.chain.Publish(ctx, message))
	s.Empty(s.subscriber1.Events())
	s.Equal([]string{message}, s.subscriber2.Events())
}

func (s *ChainedPublisherSuite) TestPublish_DontIgnoreErrors() {
	s.ignoreErrors = false
	s.subscriber1.badSubscriber = true
	s.setupChain()

	ctx := context.Background()
	message := "test"
	s.Error(s.chain.Publish(ctx, message))
	s.Empty(s.subscriber1.Events())
	s.Empty(s.subscriber2.Events())
}

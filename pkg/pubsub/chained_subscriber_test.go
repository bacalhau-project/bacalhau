package pubsub

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ChainedSubscriberSuite struct {
	suite.Suite
	chain        *ChainedSubscriber[string]
	ignoreErrors bool
	subscriber1  *InMemorySubscriber[string]
	subscriber2  *InMemorySubscriber[string]
}

func (s *ChainedSubscriberSuite) SetupTest() {
	s.subscriber1 = NewInMemorySubscriber[string]()
	s.subscriber2 = NewInMemorySubscriber[string]()

}

func (s *ChainedSubscriberSuite) setupChain() {
	s.chain = NewChainedSubscriber[string](s.ignoreErrors)
	s.chain.Add(s.subscriber1)
	s.chain.Add(s.subscriber2)
}

func TestChainedSubscriberSuite(t *testing.T) {
	suite.Run(t, new(ChainedSubscriberSuite))
}

func (s *ChainedSubscriberSuite) TestHandle_NoErrors() {
	s.ignoreErrors = false
	s.setupChain()

	ctx := context.Background()
	message := "test"
	s.NoError(s.chain.Handle(ctx, message))
	s.Equal([]string{message}, s.subscriber1.Events())
	s.Equal([]string{message}, s.subscriber2.Events())
}

func (s *ChainedSubscriberSuite) TestHandle_IgnoreErrors() {
	s.ignoreErrors = true
	s.subscriber1.badSubscriber = true
	s.setupChain()

	ctx := context.Background()
	message := "test"
	s.NoError(s.chain.Handle(ctx, message))
	s.Empty(s.subscriber1.Events())
	s.Equal([]string{message}, s.subscriber2.Events())
}

func (s *ChainedSubscriberSuite) TestHandle_DontIgnoreErrors() {
	s.ignoreErrors = false
	s.subscriber1.badSubscriber = true
	s.setupChain()

	ctx := context.Background()
	message := "test"
	s.Error(s.chain.Handle(ctx, message))
	s.Empty(s.subscriber1.Events())
	s.Empty(s.subscriber2.Events())
}

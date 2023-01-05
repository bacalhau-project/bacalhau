package pubsub

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Only the first subscriber will be considered
func TestSingletonPubSub(t *testing.T) {
	ctx := context.Background()
	s := SingletonPubSub[string]{}
	subscriber1 := NewInMemorySubscriber[string]()
	subscriber2 := NewInMemorySubscriber[string]()

	s.Subscribe(ctx, subscriber1)
	s.Subscribe(ctx, subscriber2)
	assert.Equal(t, subscriber1, s.Subscriber)
}

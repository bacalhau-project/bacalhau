package wasmlogs

import (
	"context"
	"sync/atomic"
)

type MessageQueue struct {
	ctx         context.Context
	messages    chan *Message
	counter     uint64
	channelSize int
}

func NewMessageQueue(ctx context.Context, channelSize int) *MessageQueue {
	return &MessageQueue{
		ctx:         ctx,
		messages:    make(chan *Message, channelSize),
		counter:     0,
		channelSize: channelSize,
	}
}

func (mq *MessageQueue) Enqueue(msg *Message) {
	atomic.AddUint64(&mq.counter, 1)
	mq.messages <- msg
}

func (mq *MessageQueue) Dequeue() *Message {
	select {
	case <-mq.ctx.Done():
		return nil
	case msg := <-mq.messages:
		atomic.AddUint64(&mq.counter, ^uint64(0))
		return msg
	}
}

// Full returns a boolean true if another call
// to enqueue would block (because the total count
// equals the channel buffer)
func (mq *MessageQueue) Full() bool {
	return mq.counter == uint64(mq.channelSize)
}

func (mq *MessageQueue) Count() uint64 {
	return mq.counter
}

func (mq *MessageQueue) IsEmpty() bool {
	return mq.counter == 0
}

func (mq *MessageQueue) Drain() []*Message {
	if mq.counter == 0 {
		return make([]*Message, 0)
	}

	messages := make([]*Message, mq.counter)
	for i := uint64(0); i < mq.counter; i++ {
		messages[i] = <-mq.messages
	}
	return messages
}

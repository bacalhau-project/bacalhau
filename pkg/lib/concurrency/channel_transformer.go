package concurrency

import "context"

// AsyncChannelTransform copies messages from an input channel, transforms them, and sends them to an output channel.
func AsyncChannelTransform[In any, Out any](
	ctx context.Context,
	input <-chan In,
	bufferCapacity int,
	transform func(In) (*AsyncResult[Out], error)) <-chan *AsyncResult[Out] {
	output := make(chan *AsyncResult[Out], bufferCapacity)

	go func() {
		defer close(output)
		for {
			select {
			case msg, ok := <-input:
				if !ok {
					return // Input channel closed
				}
				result := NewAsyncResult[Out](transform(msg))

				select {
				case output <- result:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return output
}

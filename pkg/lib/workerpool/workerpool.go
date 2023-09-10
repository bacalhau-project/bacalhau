package workerpool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// WorkerFunc defines a function which will be executed in a worker the pool
type WorkerFn[MsgType any] func(MsgType) error

// WorkerPool is a pool of worker threads. When given a message it puts it
// on an input channel shared by all of the worker routines, allowing one of
// them to pick it up.
type WorkerPool[MsgType any] struct {
	InputChannel chan MsgType
	workerFunc   WorkerFn[MsgType]
	workerCount  int
	wg           sync.WaitGroup
	closeChannel chan struct{}
}

type WorkerPoolConfig struct {
	workerCount      int
	inputChannelSize int
}

// NewWorkerPool creates a new worker pool which receives messages of type
// `MsgType` and allocates to the defined worker function. The worker
// function itself should do its work as quickly as possible, and if it
// needs to do any long-lived work it should initiate another goroutine
// to do the work so it can return quickly.
func NewWorkerPool[MsgType any](worker WorkerFn[MsgType], options ...OptionFn) (*WorkerPool[MsgType], error) {
	config := WorkerPoolConfig{
		workerCount:      3,
		inputChannelSize: 1,
	}

	for _, option := range options {
		option(&config)
	}

	return &WorkerPool[MsgType]{
		InputChannel: make(chan MsgType),
		workerCount:  config.workerCount,
		workerFunc:   worker,
		closeChannel: make(chan struct{}),
	}, nil
}

// Start creates and starts the worker functions ready to handle messages
func (w *WorkerPool[MsgType]) Start(ctx context.Context) {
	for i := 0; i <= w.workerCount; i++ {
		w.wg.Add(1)
		go w.runWorker(ctx, i+1, w.InputChannel)
	}
}

// Submit sends a message to the input channel, to be picked up by one of
// the workers and processed.
func (w *WorkerPool[MsgType]) Submit(m MsgType) {
	w.InputChannel <- m
}

// runWorker loops forever waiting to be told the pool is shutting down (or
// has been cancelled) or to receive a message. When given a message it will
// invoke the worker function, logging any errors it encounters.
func (w *WorkerPool[MsgType]) runWorker(ctx context.Context, id int, input chan MsgType) {
	defer w.wg.Done()

	for {
		select {
		case <-w.closeChannel:
			return
		case <-ctx.Done():
			return
		case msg := <-input:
			err := w.workerFunc(msg)
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("failed to process message in worker function")
			}
		}
	}
}

// Shutdown tells all of the workers to shutdown, and then returns, but
// will only wait for a short period of time. When shutting down, no
// attempt is made to drain the potentially buffered input channel as
// we have no way to know whether the workerfunc will be able to process
// any more messages or not.
func (w *WorkerPool[MsgType]) Shutdown(waitfor time.Duration) error {
	close(w.closeChannel)

	c := make(chan struct{})

	go func() {
		defer close(c)
		w.wg.Wait()
	}()

	select {
	case <-c:
		return nil
	case <-time.After(waitfor):
		return fmt.Errorf("shutdown timedout waiting for workers")
	}
}

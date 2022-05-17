package system

import (
	"context"
	"os"
	"os/signal"
	"sync"
)

type CancelContext struct {
	Ctx    context.Context
	Cancel context.CancelFunc
	Wg     sync.WaitGroup
}

func GetCancelContext() *CancelContext {
	ctxWithCancel, cancelFunction := context.WithCancel(context.Background())
	return &CancelContext{
		Ctx:    ctxWithCancel,
		Cancel: cancelFunction,
	}
}

// used to trigger the stop method on ctrl+c
func (cancelContext *CancelContext) HandleSignals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	cancelContext.Stop()
	os.Exit(1)
}

// add a function that will "shut down" something
// it will wait for the cancel context to be called
// then after the shutdown has been run - call done
// on the wait group
func (cancelContext *CancelContext) AddShutdownHandler(fn func()) {
	cancelContext.Wg.Add(1)
	go func(cancelContext *CancelContext, fn func()) {
		defer cancelContext.Wg.Done()
		<-cancelContext.Ctx.Done()
		fn()
	}(cancelContext, fn)
}

// this will block until all shutdown handlers are complete
func (cancelContext *CancelContext) Stop() {
	cancelContext.Cancel()
	cancelContext.Wg.Wait()
}

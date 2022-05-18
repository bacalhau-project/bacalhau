package system

import (
	"context"
	"os"
	"os/signal"
	"sync"
)

type CancelContext struct {
	Ctx            context.Context
	cancelFunction context.CancelFunc
	wg             sync.WaitGroup
}

func GetCancelContext() *CancelContext {
	ctxWithCancel, cancelFunction := context.WithCancel(context.Background())
	return &CancelContext{
		Ctx:            ctxWithCancel,
		cancelFunction: cancelFunction,
	}
}

func GetCancelContextWithSignals() *CancelContext {
	cancelContext := GetCancelContext()
	go cancelContext.HandleSignals()
	return cancelContext
}

// used to trigger the stop method on ctrl+c
func (cancelContext *CancelContext) HandleSignals() {
	signal.Reset(os.Interrupt)
	osSignalChannel := make(chan os.Signal, 1)
	signal.Notify(osSignalChannel, os.Interrupt)
	<-osSignalChannel
	cancelContext.Stop()
	os.Exit(1)
}

// add a function that will "shut down" something
// it will wait for the cancel context to be called
// then after the shutdown has been run - call done
// on the wait group
func (cancelContext *CancelContext) AddShutdownHandler(fn func()) {
	cancelContext.wg.Add(1)
	go func(cancelContext *CancelContext, fn func()) {
		defer cancelContext.wg.Done()
		<-cancelContext.Ctx.Done()
		if !ShouldKeepStack() {
			fn()
		}
	}(cancelContext, fn)
}

// this will block until all shutdown handlers are complete
func (cancelContext *CancelContext) Stop() {
	cancelContext.cancelFunction()
	cancelContext.wg.Wait()
}

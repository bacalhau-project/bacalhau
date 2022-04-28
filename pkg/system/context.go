package system

import (
	"context"
	"os"
	"os/signal"
	"time"
)

func GetCancelContext() (context.Context, context.CancelFunc) {
	ctx := context.Background()
	ctxWithCancel, cancelFunction := context.WithCancel(ctx)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			cancelFunction()
			//give things time to settle
			time.Sleep(time.Second * 1)
			os.Exit(1)
		}
	}()

	return ctxWithCancel, cancelFunction
}

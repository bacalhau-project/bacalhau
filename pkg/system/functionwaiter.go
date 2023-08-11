package system

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

type FunctionWaiter struct {
	Name        string
	MaxAttempts int
	Delay       time.Duration
	Handler     func() (bool, error)
}

func (waiter *FunctionWaiter) Wait(ctx context.Context) error {
	currentAttempts := 0

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		result, err := waiter.Handler()
		if err != nil {
			return err
		}
		if result {
			return nil
		}

		currentAttempts++
		if currentAttempts >= waiter.MaxAttempts {
			log.Ctx(ctx).Warn().Str("name", waiter.Name).Int("max", waiter.MaxAttempts).Msg("max attempts reached")
			return fmt.Errorf("%s max attempts reached: %d", waiter.Name, waiter.MaxAttempts)
		}

		time.Sleep(waiter.Delay)
	}
}

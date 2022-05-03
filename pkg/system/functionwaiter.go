package system

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

type FunctionWaiter struct {
	Name        string
	MaxAttempts int
	Delay       time.Duration
	Logging     bool
	Handler     func() (bool, error)
}

func (waiter *FunctionWaiter) Wait() error {

	currentAttempts := 0

	for {
		result, err := waiter.Handler()
		if result {
			return err
		}
		if waiter.Logging && err != nil {
			log.Debug().Msgf("waiting for %s: %s", waiter.Name, err.Error())
		}
		currentAttempts++
		if currentAttempts >= waiter.MaxAttempts {
			return fmt.Errorf("%s max attempts reached: %d", waiter.Name, waiter.MaxAttempts)
		}
		time.Sleep(waiter.Delay)
	}
}

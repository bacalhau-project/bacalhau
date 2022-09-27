package system

import (
	"fmt"
	"time"
)

type FunctionWaiter struct {
	Name        string
	MaxAttempts int
	Delay       time.Duration
	Handler     func() (bool, error)
}

func (waiter *FunctionWaiter) Wait() error {
	currentAttempts := 0

	for {
		result, err := waiter.Handler()
		if err != nil {
			return err
		}
		if result {
			return nil
		}

		currentAttempts++
		if currentAttempts >= waiter.MaxAttempts {
			return fmt.Errorf("%s max attempts reached: %d", waiter.Name, waiter.MaxAttempts)
		}

		time.Sleep(waiter.Delay)
	}
}

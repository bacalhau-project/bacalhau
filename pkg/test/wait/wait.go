package wait

import (
	"fmt"
	"time"
)

const (
	defaultRetries  = 500
	defaultInterval = 10 * time.Millisecond
)

type conditionFn func() (bool, error)

type Params struct {
	Condition conditionFn
	Retries   uint64
	Backoff   time.Duration
	FailFast  bool
}

func For(params Params) error {
	if params.Retries == 0 {
		params.Retries = defaultRetries
	}
	if params.Backoff == 0 {
		params.Backoff = defaultInterval
	}
	var lastErr error
	for tries := params.Retries; tries > 0; tries-- {
		success, err := params.Condition()
		if success {
			return nil
		}
		if err != nil && params.FailFast {
			return err
		} else {
			lastErr = err
		}
		time.Sleep(params.Backoff)
	}
	if lastErr == nil {
		return fmt.Errorf("timed out after %v", time.Duration(params.Retries)*params.Backoff)
	}
	return lastErr
}

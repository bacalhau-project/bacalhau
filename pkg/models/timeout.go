package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
)

// TimeoutConfig is the configuration for timeout related settings,
// such as execution and shutdown timeouts.
type TimeoutConfig struct {
	// ExecutionTimeout is the maximum amount of time a task is allowed to run.
	// Zero means no timeout, such as for a daemon task.
	ExecutionTimeout time.Duration
}

// Copy returns a deep copy of the timeout config.
func (t *TimeoutConfig) Copy() *TimeoutConfig {
	if t == nil {
		return nil
	}
	return &TimeoutConfig{
		ExecutionTimeout: t.ExecutionTimeout,
	}
}

func (t *TimeoutConfig) Validate() error {
	if t == nil {
		return errors.New("missing timeout config")
	}
	var mErr multierror.Error
	if t.ExecutionTimeout < 0 {
		mErr.Errors = append(mErr.Errors, fmt.Errorf("invalid execution timeout value: %s", t.ExecutionTimeout))
	}
	return mErr.ErrorOrNil()
}
